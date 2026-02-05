package tsdb

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// WriteDeviceMetric writes a single device measurement to VictoriaMetrics.
//
// This is the primary method for recording device telemetry data.
// The write is non-blocking; data is batched and sent asynchronously.
//
// Parameters:
//   - deviceID: Unique identifier for the device (e.g., "light-living-01")
//   - measurement: The metric name (e.g., "power_watts", "temperature")
//   - value: The numeric value to record
//
// Example:
//
//	client.WriteDeviceMetric("thermostat-01", "temperature", 21.5)
//	client.WriteDeviceMetric("light-kitchen", "power_watts", 23.0)
func (c *Client) WriteDeviceMetric(deviceID string, measurement string, value float64) {
	c.addLine(formatLineProtocol(
		"device_metrics",
		map[string]string{
			"device_id":   deviceID,
			"measurement": measurement,
		},
		map[string]interface{}{
			"value": value,
		},
		time.Now(),
	))
}

// WriteEnergyMetric writes an energy consumption measurement.
//
// Used for tracking power usage and energy efficiency.
//
// Parameters:
//   - deviceID: Device identifier
//   - powerWatts: Current power draw in watts
//   - energyKWh: Cumulative energy consumption in kWh (use 0 if unknown)
func (c *Client) WriteEnergyMetric(deviceID string, powerWatts float64, energyKWh float64) {
	fields := map[string]interface{}{
		"power_watts": powerWatts,
	}
	if energyKWh > 0 {
		fields["energy_kwh"] = energyKWh
	}

	c.addLine(formatLineProtocol(
		"energy",
		map[string]string{
			"device_id": deviceID,
		},
		fields,
		time.Now(),
	))
}

// WritePHMMetric writes a Predictive Health Monitoring measurement.
//
// Used for tracking device health indicators like runtime hours,
// cycle counts, and anomaly scores.
//
// Parameters:
//   - deviceID: Device identifier
//   - metricName: PHM metric (e.g., "runtime_hours", "cycle_count", "anomaly_score")
//   - value: The metric value
func (c *Client) WritePHMMetric(deviceID string, metricName string, value float64) {
	c.addLine(formatLineProtocol(
		"phm",
		map[string]string{
			"device_id": deviceID,
			"metric":    metricName,
		},
		map[string]interface{}{
			"value": value,
		},
		time.Now(),
	))
}

// WritePoint writes a custom point with full control over tags and fields.
//
// Use this for custom measurements that don't fit the helper methods.
//
// Parameters:
//   - measurement: The measurement name (table)
//   - tags: Key-value pairs for indexing (low cardinality)
//   - fields: Key-value pairs for the actual data
//
// Example:
//
//	client.WritePoint("system_stats",
//	    map[string]string{"host": "core-01"},
//	    map[string]interface{}{"cpu_percent": 45.2, "memory_mb": 512})
func (c *Client) WritePoint(measurement string, tags map[string]string, fields map[string]interface{}) {
	c.addLine(formatLineProtocol(measurement, tags, fields, time.Now()))
}

// WritePointWithTime writes a custom point with a specific timestamp.
//
// Use this when the timestamp is not "now" (e.g., delayed data).
//
// Parameters:
//   - measurement: The measurement name
//   - tags: Key-value pairs for indexing
//   - fields: Key-value pairs for the data
//   - timestamp: The exact time for this data point
func (c *Client) WritePointWithTime(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) {
	c.addLine(formatLineProtocol(measurement, tags, fields, timestamp))
}

// formatLineProtocol formats a data point as an InfluxDB line protocol string.
//
// Format: measurement,tag1=val1,tag2=val2 field1=val1,field2=val2 timestamp_ns
//
// VictoriaMetrics accepts this format on the /write endpoint.
func formatLineProtocol(measurement string, tags map[string]string, fields map[string]interface{}, t time.Time) string {
	var b strings.Builder

	// Measurement (escaped to prevent injection)
	b.WriteString(escapeMeasurement(measurement))

	// Tags (sorted for deterministic output and testability)
	tagKeys := make([]string, 0, len(tags))
	for k := range tags {
		tagKeys = append(tagKeys, k)
	}
	sort.Strings(tagKeys)
	for _, k := range tagKeys {
		b.WriteByte(',')
		b.WriteString(escapeTag(k))
		b.WriteByte('=')
		b.WriteString(escapeTag(tags[k]))
	}

	// Fields (sorted for deterministic output)
	fieldKeys := make([]string, 0, len(fields))
	for k := range fields {
		fieldKeys = append(fieldKeys, k)
	}
	sort.Strings(fieldKeys)
	b.WriteByte(' ')
	first := true
	for _, k := range fieldKeys {
		v := fields[k]
		if !first {
			b.WriteByte(',')
		}
		first = false
		b.WriteString(escapeTag(k))
		b.WriteByte('=')
		switch val := v.(type) {
		case float64:
			b.WriteString(fmt.Sprintf("%g", val))
		case int:
			b.WriteString(fmt.Sprintf("%di", val))
		case int64:
			b.WriteString(fmt.Sprintf("%di", val))
		case bool:
			if val {
				b.WriteString("true")
			} else {
				b.WriteString("false")
			}
		case string:
			b.WriteString(fmt.Sprintf("%q", val))
		default:
			b.WriteString(fmt.Sprintf("%v", val))
		}
	}

	// Timestamp in nanoseconds
	b.WriteByte(' ')
	b.WriteString(fmt.Sprintf("%d", t.UnixNano()))

	return b.String()
}

// escapeTag escapes special characters in tag keys/values per line protocol spec.
// Commas, equals signs, and spaces must be backslash-escaped.
// Newlines are stripped to prevent line protocol injection.
func escapeTag(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, " ", "\\ ")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "=", "\\=")
	return s
}

// escapeMeasurement escapes special characters in measurement names.
// Newlines are stripped to prevent line protocol injection.
func escapeMeasurement(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, " ", "\\ ")
	s = strings.ReplaceAll(s, ",", "\\,")
	return s
}
