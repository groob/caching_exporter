package main

import (
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"howett.net/plist"
)

const (
	configPlistPath    = "/Library/Server/Caching/Config/Config.plist"
	lastStatePlistPath = "/Library/Server/Caching/Logs/LastState.plist"
)

var (
	// Config.plist metrics
	cacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "caching_saved_cache_size",
		Help: "SavedCacheSize from Config.plist",
	})
	reservedVolumeSpace = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "caching_reserved_volume_space",
		Help: "ReservedVolumeSpace from Config.plist",
	})
	cachingData = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "caching_data",
		Help: "data cached by server.",
	},
		[]string{
			"type",
		})

	// LastState.plist metrics
	active = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "caching_status_active",
		Help: "whether caching server is currently running",
	})
	peers = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "caching_peers_total",
		Help: "Number of Caching Server peers",
	})
	bytesFromOrigin = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "caching_bytes_from_origin_total",
		Help: "Number of bytes returned from origin",
	})
	bytesFromPeers = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "caching_bytes_from_peers_total",
		Help: "Number of bytes returned from peers",
	})
	bytesRequested = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "caching_bytes_requested_total",
		Help: "Number of bytes requested",
	})
	bytesReturned = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "caching_bytes_returned_total",
		Help: "Number of bytes returned",
	})
)

func init() {
	// Config.plist metrics
	prometheus.MustRegister(cacheSize)
	prometheus.MustRegister(reservedVolumeSpace)
	prometheus.MustRegister(cachingData)
	// LastState.plist metrics
	prometheus.MustRegister(active)
	prometheus.MustRegister(peers)
	prometheus.MustRegister(bytesFromOrigin)
	prometheus.MustRegister(bytesFromPeers)
	prometheus.MustRegister(bytesRequested)
	prometheus.MustRegister(bytesReturned)
}

type configPlist struct {
	LastRegOrFlush      *time.Time
	SavedCacheSize      int
	ReservedVolumeSpace int
	SavedCacheDetails   struct {
		IOSSoftware int `plist:"iOS Software"`
		MacSoftware int `plist:"Mac Software"`
		ICloud      int `plist:"iCloud"`
		Books       int `plist:"Books"`
		ITunesU     int `plist:"iTunes U"`
		Movies      int `plist:"Movies"`
		Music       int `plist:"Music"`
		Other       int `plist:"Other"`
	}
}

type lastStatePlist struct {
	Active               bool
	Peers                []string
	CacheFree            int
	CacheLimit           int
	CacheStatus          string
	StartupStatus        string
	State                string `plist:"state"`
	CacheUsed            int
	RegistrationStatus   int
	TotalBytesFromOrigin int
	TotalBytesFromPeers  int
	TotalBytesRequested  int
	TotalBytesReturned   int
}

func (p *configPlist) parse() error {
	f, err := os.Open(configPlistPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return plist.NewDecoder(f).Decode(p)
}

// The LastState.plist file only appears if Caching server is running and the
// serveradmin fullstatus caching command runs
// this also requers caching_exporter to be run as root
func checkCachingStatus() error {
	cmd := exec.Command("/Applications/Server.app/Contents/ServerRoot/usr/sbin/serveradmin", "fullstatus", "caching")
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	return cmd.Wait()
}
func (p *lastStatePlist) parse() error {
	err := checkCachingStatus()
	if err != nil {
		return err
	}
	// if the file is not present, the file won't exist
	// return without errors
	if _, err := os.Stat(lastStatePlistPath); os.IsNotExist(err) {
		return nil
	}
	f, err := os.Open(lastStatePlistPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return plist.NewDecoder(f).Decode(p)
}

func (p lastStatePlist) setMetrics() error {
	err := p.parse()
	if err != nil {
		return err
	}
	if p.Active {
		active.Set(1)
		peers.Set(float64(len(p.Peers)))
		bytesFromOrigin.Set(float64(p.TotalBytesFromOrigin))
		bytesFromPeers.Set(float64(p.TotalBytesFromPeers))
		bytesRequested.Set(float64(p.TotalBytesRequested))
		bytesReturned.Set(float64(p.TotalBytesReturned))
	} else {
		active.Set(0)
		peers.Set(0)
	}
	return nil
}

func (p configPlist) setMetrics() error {
	err := p.parse()
	if err != nil {
		return err
	}
	cacheSize.Set(float64(p.SavedCacheSize))
	reservedVolumeSpace.Set(float64(p.ReservedVolumeSpace))
	cachingData.WithLabelValues("iOS Software").Set(float64(p.SavedCacheDetails.IOSSoftware))
	cachingData.WithLabelValues("Mac Software").Set(float64(p.SavedCacheDetails.MacSoftware))
	cachingData.WithLabelValues("iCloud").Set(float64(p.SavedCacheDetails.ICloud))
	cachingData.WithLabelValues("Books").Set(float64(p.SavedCacheDetails.Books))
	cachingData.WithLabelValues("iTunesU").Set(float64(p.SavedCacheDetails.ITunesU))
	cachingData.WithLabelValues("Movies").Set(float64(p.SavedCacheDetails.Movies))
	cachingData.WithLabelValues("Music").Set(float64(p.SavedCacheDetails.Music))
	cachingData.WithLabelValues("Other").Set(float64(p.SavedCacheDetails.Other))

	return nil
}

func monitor() {
	ticker := time.NewTicker(time.Second * 30).C
	var config configPlist
	var lastState lastStatePlist
	for {
		// read data from Config.plist
		err := config.setMetrics()
		if err != nil {
			log.Println(err)
			continue
		}
		// read data from LastState.plist
		err = lastState.setMetrics()
		if err != nil {
			log.Println(err)
			continue
		}
		<-ticker
	}
}
