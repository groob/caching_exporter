// Copyright 2011 Google Inc. All Rights Reserved.
// This file is available under the Apache license.

package main

import (
	"flag"
	"log"
	"strings"

	"github.com/golang/glog"
	"github.com/google/mtail/mtail"
	"github.com/prometheus/client_golang/prometheus"

	"net/http"
	_ "net/http/pprof"
)

var (
	port  = flag.String("port", "3903", "HTTP port to listen on.")
	logs  = flag.String("logs", "", "List of files to monitor.")
	progs = flag.String("progs", "", "Directory containing programs")
	// only used by the mtail collector Describe()
	collected = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "caching_collected_metrics",
		Help: "total collected metrics",
	})
)

func main() {
	flag.Parse()
	if *progs == "" {
		glog.Exitf("No mtail program directory specified; use -progs")
	}
	if *logs == "" {
		glog.Exitf("No logs specified to tail; use -logs")
	}
	var logPathnames []string
	for _, pathname := range strings.Split(*logs, ",") {
		if pathname != "" {
			logPathnames = append(logPathnames, pathname)
		}
	}
	if len(logPathnames) == 0 {
		glog.Exit("No logs to tail.")
	}
	o := mtail.Options{
		Progs:    *progs,
		LogPaths: logPathnames,
		Port:     *port,
	}
	m, err := mtail.New(o)
	if err != nil {
		glog.Fatalf("couldn't start: %s", err)
	}

	c := newMtailCollector(m)
	prometheus.MustRegister(c)

	go monitor()

	http.Handle("/metrics", prometheus.Handler())
	log.Fatal(http.ListenAndServe(":"+*port, nil))

}
