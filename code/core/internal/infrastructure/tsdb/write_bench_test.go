package tsdb

import (
	"testing"
	"time"
)

func BenchmarkFormatLineProtocol_Simple(b *testing.B) {
	tags := map[string]string{"device_id": "light-01", "measurement": "power_watts"}
	fields := map[string]interface{}{"value": 23.5}
	ts := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatLineProtocol("device_metrics", tags, fields, ts)
	}
}

func BenchmarkFormatLineProtocol_MultiField(b *testing.B) {
	tags := map[string]string{"device_id": "thermostat-01"}
	fields := map[string]interface{}{
		"temperature": 21.5,
		"humidity":    45.0,
		"setpoint":    22.0,
		"mode":        "heating",
	}
	ts := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatLineProtocol("climate", tags, fields, ts)
	}
}

func BenchmarkFormatLineProtocol_ManyTags(b *testing.B) {
	tags := map[string]string{
		"device_id": "light-living-01",
		"protocol":  "knx",
		"domain":    "lighting",
		"room":      "living-room",
		"area":      "ground-floor",
	}
	fields := map[string]interface{}{"value": 75.0}
	ts := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatLineProtocol("device_metrics", tags, fields, ts)
	}
}

func BenchmarkEscapeTag(b *testing.B) {
	for i := 0; i < b.N; i++ {
		escapeTag("device_id=light,room 01")
	}
}
