// Package knx implements the KNX protocol bridge for Gray Logic.
//
// This package provides connectivity to KNX building automation systems via
// the knxd daemon. It translates between Gray Logic's internal representation
// and KNX group address telegrams.
//
// # Architecture
//
// The bridge operates as a translator between two buses:
//
//	┌─────────────────┐          ┌─────────────────┐
//	│   Gray Logic    │   MQTT   │   KNX Bridge    │   knxd
//	│      Core       │◄────────►│   (this pkg)    │◄────────► KNX Bus
//	└─────────────────┘          └─────────────────┘
//
// # Key Responsibilities
//
//   - Connect to knxd daemon via Unix socket or TCP
//   - Subscribe to KNX group address telegrams
//   - Translate KNX telegrams to MQTT state messages
//   - Translate MQTT commands to KNX telegrams
//   - Handle DPT (Datapoint Type) encoding/decoding
//   - Publish health status and metrics
//
// # Group Addresses
//
// KNX uses group addresses for communication. This package uses the 3-level
// format: Main/Middle/Sub (e.g., "1/2/3").
//
// Example:
//
//	addr, err := knx.ParseGroupAddress("1/2/3")
//	if err != nil {
//	    return err
//	}
//	fmt.Println(addr.String()) // "1/2/3"
//
// # Datapoint Types
//
// KNX defines standardised data formats (DPTs). This package supports common
// DPTs for lighting, blinds, climate, and sensors:
//
//   - DPT 1.xxx: 1-bit (switch, bool, up/down)
//   - DPT 5.xxx: 1-byte unsigned (percentage, angle)
//   - DPT 9.xxx: 2-byte float (temperature, lux)
//   - DPT 232.600: 3-byte RGB colour
//
// # Thread Safety
//
// All exported types are safe for concurrent use from multiple goroutines.
//
// # References
//
//   - KNX Specification: https://www.knx.org
//   - knxd daemon: https://github.com/knxd/knxd
//   - Gray Logic KNX spec: docs/protocols/knx.md
package knx
