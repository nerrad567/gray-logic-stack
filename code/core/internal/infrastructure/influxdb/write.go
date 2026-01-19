package influxdb

import (
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// WriteDeviceMetric writes a single device measurement to InfluxDB.
//
// This is the primary method for recording device telemetry data.
// The write is non-blocking; data is batched and sent asynchronously.
//
// Parameters:
//   - deviceID: Unique identifier for the device (e.g., "light-living-01")
//   - measurement: The metric name (e.g., "power_watts", "temperature_c")
//   - value: The numeric value to record
//
// Example:
//
//	client.WriteDeviceMetric("thermostat-01", "temperature_c", 21.5)
//	client.WriteDeviceMetric("light-kitchen", "power_watts", 23.0)
func (c *Client) WriteDeviceMetric(deviceID string, measurement string, value float64) {
	if !c.IsConnected() {
		return
	}

	point := write.NewPoint(
		"device_metrics",
		map[string]string{
			"device_id":   deviceID,
			"measurement": measurement,
		},
		map[string]interface{}{
			"value": value,
		},
		time.Now(),
	)

	c.writeAPI.WritePoint(point)
}

// WriteEnergyMetric writes an energy consumption measurement.
//
// Used for tracking power usage and energy efficiency.
//
// Parameters:
//   - deviceID: Device identifier
//   - powerWatts: Current power draw in watts
//   - energyKWh: Cumulative energy consumption in kWh (optional, use 0 if unknown)
func (c *Client) WriteEnergyMetric(deviceID string, powerWatts float64, energyKWh float64) {
	if !c.IsConnected() {
		return
	}

	fields := map[string]interface{}{
		"power_watts": powerWatts,
	}
	if energyKWh > 0 {
		fields["energy_kwh"] = energyKWh
	}

	point := write.NewPoint(
		"energy",
		map[string]string{
			"device_id": deviceID,
		},
		fields,
		time.Now(),
	)

	c.writeAPI.WritePoint(point)
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
	if !c.IsConnected() {
		return
	}

	point := write.NewPoint(
		"phm",
		map[string]string{
			"device_id": deviceID,
			"metric":    metricName,
		},
		map[string]interface{}{
			"value": value,
		},
		time.Now(),
	)

	c.writeAPI.WritePoint(point)
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
	if !c.IsConnected() {
		return
	}

	point := write.NewPoint(measurement, tags, fields, time.Now())
	c.writeAPI.WritePoint(point)
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
	if !c.IsConnected() {
		return
	}

	point := write.NewPoint(measurement, tags, fields, timestamp)
	c.writeAPI.WritePoint(point)
}
