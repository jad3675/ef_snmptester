// Package main is the entry point for the SNMP Tester application
package main

import (
	"fmt"
	"os"
	
	"github.com/jad3675/ef_snmptester/internal/ui"
	"github.com/spf13/pflag"
)

func main() {
	// Parse command line flags
	configPath := pflag.StringP("config", "c", "/etc/elastiflow/snmp", "Path to ElastiFlow SNMP configuration directory")
	help := pflag.BoolP("help", "h", false, "Show help")
	version := pflag.BoolP("version", "v", false, "Show version")
	pflag.Parse()
	
	// Show help if requested
	if *help {
		fmt.Println("ElastiFlow SNMP Tester")
		fmt.Println("A TUI application for testing ElastiFlow SNMP configurations")
		fmt.Println()
		pflag.PrintDefaults()
		os.Exit(0)
	}
	
	// Show version if requested
	if *version {
		fmt.Println("ElastiFlow SNMP Tester v0.1.0")
		os.Exit(0)
	}
	
	// Create and start the UI application
	app := ui.NewApp(*configPath)
	if err := app.Start(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
