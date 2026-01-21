// Package knxd provides management of the knxd daemon process.
//
// knxd (KNX daemon) is the interface between Gray Logic and the KNX bus.
// This package manages knxd as a subprocess of Gray Logic, providing:
//
//   - Configuration-driven startup (no manual /etc/knxd.conf editing)
//   - Automatic restart on failure
//   - Health monitoring
//   - Graceful shutdown coordination
//
// The knxd process is started with command-line arguments derived from
// Gray Logic's YAML configuration, eliminating the need for engineers
// to manually edit system configuration files.
//
// Example configuration (in config.yaml):
//
//	protocols:
//	  knx:
//	    enabled: true
//	    knxd:
//	      managed: true
//	      binary: "/usr/bin/knxd"
//	      physical_address: "0.0.1"
//	      client_addresses: "0.0.2:8"
//	      backend:
//	        type: "usb"
//
// When the Gray Logic Core starts, it will spawn knxd with the appropriate
// arguments and monitor it throughout the application lifecycle.
package knxd
