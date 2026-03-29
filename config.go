package server

import (
	pcolor "github.com/clong1995/go-ansi-color"
	conf "github.com/clong1995/go-config"
)

var machineID int

func config() {
	// MACHINE ID
	var exists bool
	machineID, exists = conf.Value[int]("MACHINE ID")
	if !exists || machineID == 0 {
		pcolor.PrintFatal(prefix, "MACHINE not found")
	}

}
