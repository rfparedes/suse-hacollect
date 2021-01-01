package main

import (
	"flag"
)

// Config will be the holder for our flags
type Config struct {
	version  bool
	caseNum  string
	fromDate string
	upload   bool
}

// Setup initializes a config from flags that are passed in
func (c *Config) Setup() {
	flag.BoolVar(&c.version, "v", false, "Output version information")
	flag.StringVar(&c.caseNum, "c", "", "Add CASENUM to tarball filename")
	flag.StringVar(&c.fromDate, "f", "", "Gather data for issue starting from date, using format: yyyy-mm-dd")
	flag.BoolVar(&c.upload, "u", false, "Automatically upload to SUSE via https.  (https port 443 outbound required)")
}
