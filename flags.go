package main

import "flag"

var (
	flagVersion bool
)

func initFlags() {
	flag.BoolVar(&flagVersion, "v", false, "Show version")
}
