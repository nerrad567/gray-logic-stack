// Package etsimport provides parsing and import of KNX ETS project files.
//
// ETS (Engineering Tool Software) is the standard configuration tool for KNX
// installations. This package parses .knxproj files (and XML/CSV exports) to
// extract group addresses, device configurations, and building structure.
//
// # Supported Formats
//
//   - .knxproj: Native ETS project file (ZIP archive with XML)
//   - .xml: ETS group address XML export
//   - .csv: ETS group address CSV export
//
// # Usage
//
//	parser := etsimport.NewParser()
//	result, err := parser.ParseFile("project.knxproj")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Review detected devices
//	for _, dev := range result.Devices {
//	    fmt.Printf("%s: %s (confidence: %.0f%%)\n",
//	        dev.SuggestedID, dev.DetectedType, dev.Confidence*100)
//	}
//
// # Device Detection
//
// The parser uses heuristic rules to group related group addresses into
// logical devices. For example, a dimmer is detected when both a switch
// GA (DPT 1.001) and brightness GA (DPT 5.001) share a common name prefix.
//
// Detection confidence is scored:
//   - High (>80%): Strong pattern match, auto-selected for import
//   - Medium (50-80%): Partial match, flagged for review
//   - Low (<50%): Uncertain, requires manual confirmation
//
// # Integration
//
// After parsing, the result can be passed to the import API which:
//   - Creates device records in the database
//   - Generates knx-bridge.yaml configuration
//   - Optionally creates rooms and areas from ETS building structure
package etsimport
