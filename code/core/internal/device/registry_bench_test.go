package device

import (
	"context"
	"fmt"
	"testing"
)

// setupBenchRegistry creates a registry pre-populated with n devices.
func setupBenchRegistry(b *testing.B, n int) *Registry {
	b.Helper()
	repo := NewMockRepository()
	ctx := context.Background()

	for i := 0; i < n; i++ {
		protocol := ProtocolKNX
		if i%3 == 0 {
			protocol = ProtocolMQTT
		}
		dev := &Device{
			ID:           fmt.Sprintf("dev-%04d", i),
			Name:         fmt.Sprintf("Device %d", i),
			Type:         DeviceTypeLightDimmer,
			Domain:       DomainLighting,
			Protocol:     protocol,
			Capabilities: []Capability{CapOnOff, CapDim},
			Address:      Address{"ga": fmt.Sprintf("1/%d/%d", i/256, i%256)},
			HealthStatus: HealthStatusOnline,
		}
		if err := repo.Create(ctx, dev); err != nil {
			b.Fatalf("creating device %d: %v", i, err)
		}
	}

	reg := NewRegistry(repo)
	if err := reg.RefreshCache(ctx); err != nil {
		b.Fatalf("refreshing cache: %v", err)
	}
	return reg
}

func BenchmarkRegistryGetDevice(b *testing.B) {
	reg := setupBenchRegistry(b, 100)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.GetDevice(ctx, "dev-0050") //nolint:errcheck // benchmark
	}
}

func BenchmarkRegistryGetDevice_Parallel(b *testing.B) {
	reg := setupBenchRegistry(b, 100)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			reg.GetDevice(ctx, "dev-0050") //nolint:errcheck // benchmark
		}
	})
}

func BenchmarkRegistrySetDeviceState(b *testing.B) {
	reg := setupBenchRegistry(b, 100)
	ctx := context.Background()
	state := State{"brightness": 75.0, "on": true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.SetDeviceState(ctx, "dev-0050", state) //nolint:errcheck // benchmark
	}
}

func BenchmarkRegistryGetDevicesByProtocol(b *testing.B) {
	reg := setupBenchRegistry(b, 200)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.GetDevicesByProtocol(ctx, ProtocolKNX) //nolint:errcheck // benchmark
	}
}

func BenchmarkRegistryRefreshCache(b *testing.B) {
	repo := NewMockRepository()
	ctx := context.Background()

	for i := 0; i < 200; i++ {
		dev := &Device{
			ID:       fmt.Sprintf("dev-%04d", i),
			Name:     fmt.Sprintf("Device %d", i),
			Type:     DeviceTypeLightDimmer,
			Domain:   DomainLighting,
			Protocol: ProtocolKNX,
		}
		if err := repo.Create(ctx, dev); err != nil {
			b.Fatalf("creating device %d: %v", i, err)
		}
	}

	reg := NewRegistry(repo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.RefreshCache(ctx) //nolint:errcheck // benchmark
	}
}
