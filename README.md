# caching exporter ->
# Expose OS X Caching Server metrics for collection by [Prometheus](http://prometheus.io/)

Caching exporter is modified version of [mtail](https://github.com/google/mtail) with several additions to collect metrics stored in a plist file.
Caching Exporter collects metrics from the following locations: 

* /Library/Server/Caching/Config/Config.plist
* /Library/Server/Caching/Logs/LastState.plist
* /Library/Server/Caching/Logs/Debug.log

# Usage

```
sudo ./caching_exporter -progs ./progs --logs /Library/Server/Caching/Logs/Debug.log
```
Now point Prometheus at localhost:3903/metrics to collect the data exposed by Caching exporter.

# What are progs?
Progs are files that contain regex rules for parsing data from log files into time series.
This is the functionality enabled by [mtail](https://github.com/google/mtail) and can be extended by rewriting or adding our own rules in the progs folder. 
An example:
```
# simple line counter
counter line_count
/$/ {
    line_count++
}
```
