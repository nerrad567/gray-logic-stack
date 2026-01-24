package knxd

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// BackendType represents how knxd connects to the KNX bus.
type BackendType string

const (
	// BackendUSB connects via a USB KNX interface (e.g., Weinzierl).
	BackendUSB BackendType = "usb"

	// BackendIPTunnel connects via KNX/IP tunnelling to a gateway.
	BackendIPTunnel BackendType = "ipt"

	// BackendIPRouting connects via KNX/IP routing (multicast).
	BackendIPRouting BackendType = "ip"
)

// Config holds the configuration for the knxd daemon.
type Config struct {
	// Managed indicates whether Gray Logic should manage knxd lifecycle.
	// If false, knxd is expected to be running externally.
	Managed bool `yaml:"managed"`

	// Binary is the path to the knxd executable.
	// Default: "/usr/bin/knxd"
	Binary string `yaml:"binary"`

	// PhysicalAddress is knxd's own address on the KNX bus.
	// Format: "area.line.device" (e.g., "0.0.1")
	// This is the -e flag for knxd.
	PhysicalAddress string `yaml:"physical_address"`

	// ClientAddresses is the range of addresses knxd can assign to clients.
	// Format: "area.line.device:count" (e.g., "0.0.2:8" means 0.0.2 through 0.0.9)
	// This is the -E flag for knxd.
	ClientAddresses string `yaml:"client_addresses"`

	// Backend configures how knxd connects to the KNX bus.
	Backend BackendConfig `yaml:"backend"`

	// ListenTCP enables TCP listening for clients (like Gray Logic's KNX bridge).
	// Default: true (listens on 6720)
	ListenTCP bool `yaml:"listen_tcp"`

	// TCPPort is the port for TCP connections.
	// Default: 6720
	TCPPort int `yaml:"tcp_port"`

	// UnixSocket enables Unix socket listening.
	// Default: "/tmp/eib"
	UnixSocket string `yaml:"unix_socket"`

	// RestartOnFailure enables automatic restart if knxd crashes.
	// Default: true
	RestartOnFailure bool `yaml:"restart_on_failure"`

	// RestartDelay is the time to wait before restarting.
	// Default: 5s
	RestartDelay time.Duration `yaml:"restart_delay"`

	// MaxRestartAttempts limits restart attempts. 0 means unlimited.
	// Default: 10
	MaxRestartAttempts int `yaml:"max_restart_attempts"`

	// GracefulTimeout is how long to wait for graceful shutdown.
	// Default: 10s
	GracefulTimeout time.Duration `yaml:"graceful_timeout"`

	// HealthCheckInterval is how often to run watchdog health checks.
	// If knxd hangs (stops responding), it will be killed and restarted
	// after 3 consecutive health check failures.
	// Default: 30s
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`

	// HealthCheckDeviceAddress is an optional KNX group address to read during
	// health checks. This provides end-to-end verification that the entire
	// communication chain works (knxd → interface → bus → device → response).
	//
	// Recommended: Use a PSU diagnostic address or any device that reliably
	// responds to read requests.
	//
	// Format: "main/middle/sub" (e.g., "1/7/0" for PSU status)
	// If empty, bus-level health checks are disabled.
	HealthCheckDeviceAddress string `yaml:"health_check_device_address,omitempty"`

	// HealthCheckDeviceTimeout is how long to wait for a response from the
	// health check device. Should be longer than typical bus response time.
	// Default: 3s
	HealthCheckDeviceTimeout time.Duration `yaml:"health_check_device_timeout,omitempty"`

	// GroupCache enables knxd's group communication cache (-c flag).
	// This is required for routing group telegrams between local clients
	// and the backend (e.g., IPT tunnelling to a gateway).
	// Default: false
	GroupCache bool `yaml:"group_cache"`

	// LogLevel sets knxd's verbosity.
	// Range: 0 (minimal) to 9 (maximum debug)
	// Default: 0
	LogLevel int `yaml:"log_level"`

	// TraceFlags sets knxd's trace flags bitmask.
	// Default: 0 (no tracing)
	TraceFlags int `yaml:"trace_flags"`
}

// BackendConfig configures the KNX bus connection.
type BackendConfig struct {
	// Type is the backend connection type.
	// Options: "usb", "ipt", "ip"
	Type BackendType `yaml:"type"`

	// Host is the IP address for ipt (tunnelling) connections.
	// Only used when Type is "ipt".
	Host string `yaml:"host,omitempty"`

	// Port is the port for ipt connections.
	// Default: 3671
	Port int `yaml:"port,omitempty"`

	// MulticastAddress is the multicast group for ip (routing) connections.
	// Default: "224.0.23.12"
	// Only used when Type is "ip".
	MulticastAddress string `yaml:"multicast_address,omitempty"`

	// Interface is the network interface for multicast.
	// Only used when Type is "ip".
	Interface string `yaml:"interface,omitempty"`

	// USBDevice is a specific USB device path (optional).
	// If empty, knxd auto-detects the USB interface.
	// Only used when Type is "usb".
	USBDevice string `yaml:"usb_device,omitempty"`

	// USBVendorID is the vendor ID for USB reset operations.
	// Format: "0e77" (hex without 0x prefix)
	// Used by usbreset utility to identify the device.
	// Only used when Type is "usb".
	USBVendorID string `yaml:"usb_vendor_id,omitempty"`

	// USBProductID is the product ID for USB reset operations.
	// Format: "0104" (hex without 0x prefix)
	// Used by usbreset utility to identify the device.
	// Only used when Type is "usb".
	USBProductID string `yaml:"usb_product_id,omitempty"`

	// USBResetOnRetry enables USB device reset before retry attempts.
	// This helps recover from LIBUSB_ERROR_BUSY conditions.
	// Requires usbreset utility and proper udev rules.
	// Only used when Type is "usb".
	USBResetOnRetry bool `yaml:"usb_reset_on_retry,omitempty"`

	// USBResetOnBusFailure enables USB device reset when bus-level health
	// checks fail (Layer 4). This provides proactive recovery for USB
	// interface issues without waiting for the full watchdog restart cycle.
	// Requires usbreset utility and proper udev rules.
	// Only used when Type is "usb".
	USBResetOnBusFailure bool `yaml:"usb_reset_on_bus_failure,omitempty"`
}

// DefaultConfig returns a Config with sensible defaults for USB connection.
func DefaultConfig() Config {
	return Config{
		Managed:            true,
		Binary:             "/usr/bin/knxd",
		PhysicalAddress:    "0.0.1",
		ClientAddresses:    "0.0.2:8",
		ListenTCP:          true,
		TCPPort:            6720,
		UnixSocket:         "/tmp/eib",
		RestartOnFailure:   true,
		RestartDelay:       5 * time.Second,
		MaxRestartAttempts: 10,
		GracefulTimeout:    10 * time.Second,
		LogLevel:           0,
		TraceFlags:         0,
		Backend: BackendConfig{
			Type: BackendUSB,
		},
	}
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if c.Binary == "" {
		return fmt.Errorf("knxd binary path is required")
	}

	if err := validateKNXAddress(c.PhysicalAddress); err != nil {
		return fmt.Errorf("invalid physical_address: %w", err)
	}

	if err := validateClientAddresses(c.ClientAddresses); err != nil {
		return fmt.Errorf("invalid client_addresses: %w", err)
	}

	if err := c.Backend.Validate(); err != nil {
		return fmt.Errorf("invalid backend config: %w", err)
	}

	if c.TCPPort < 1 || c.TCPPort > 65535 {
		return fmt.Errorf("tcp_port must be between 1 and 65535")
	}

	if c.LogLevel < 0 || c.LogLevel > 9 {
		return fmt.Errorf("log_level must be between 0 and 9")
	}

	// Validate health check device address if specified
	if c.HealthCheckDeviceAddress != "" {
		if _, err := ParseGroupAddress(c.HealthCheckDeviceAddress); err != nil {
			return fmt.Errorf("invalid health_check_device_address: %w", err)
		}
	}

	return nil
}

// Validate checks the backend configuration.
func (b *BackendConfig) Validate() error {
	switch b.Type {
	case BackendUSB:
		// Validate USBDevice if specified (prevent command injection)
		if b.USBDevice != "" {
			if err := validateSafePathComponent(b.USBDevice, "usb_device"); err != nil {
				return err
			}
		}
		// Validate USB IDs if specified (hex format, 4 chars)
		if b.USBVendorID != "" {
			if err := validateUSBID(b.USBVendorID, "usb_vendor_id"); err != nil {
				return err
			}
		}
		if b.USBProductID != "" {
			if err := validateUSBID(b.USBProductID, "usb_product_id"); err != nil {
				return err
			}
		}
		// If any USB reset option is enabled, both IDs must be specified
		if b.USBResetOnRetry && (b.USBVendorID == "" || b.USBProductID == "") {
			return fmt.Errorf("usb_reset_on_retry requires both usb_vendor_id and usb_product_id")
		}
		if b.USBResetOnBusFailure && (b.USBVendorID == "" || b.USBProductID == "") {
			return fmt.Errorf("usb_reset_on_bus_failure requires both usb_vendor_id and usb_product_id")
		}
		return nil

	case BackendIPTunnel:
		if b.Host == "" {
			return fmt.Errorf("host is required for ipt backend")
		}
		// Skip DNS resolution check — the host may not be resolvable yet
		// (e.g., Docker service names resolve only after container startup).
		// knxd itself will handle resolution at connect time.
		return nil

	case BackendIPRouting:
		if b.MulticastAddress == "" {
			b.MulticastAddress = "224.0.23.12" // KNX default multicast
		}
		ip := net.ParseIP(b.MulticastAddress)
		if ip == nil || !ip.IsMulticast() {
			return fmt.Errorf("invalid multicast_address %q", b.MulticastAddress)
		}
		// Validate Interface if specified (prevent command injection)
		if b.Interface != "" {
			if err := validateSafePathComponent(b.Interface, "interface"); err != nil {
				return err
			}
		}
		return nil

	case "":
		return fmt.Errorf("backend type is required")

	default:
		return fmt.Errorf("unknown backend type %q (use: usb, ipt, ip)", b.Type)
	}
}

// BuildArgs constructs the command-line arguments for knxd.
func (c *Config) BuildArgs() []string {
	var args []string

	// Physical address (-e)
	args = append(args, "-e", c.PhysicalAddress)

	// Client address pool (-E)
	args = append(args, "-E", c.ClientAddresses)

	// Unix socket (-u)
	if c.UnixSocket != "" {
		args = append(args, "-u", c.UnixSocket)
	}

	// TCP server (-i) - required for Gray Logic bridge to connect
	// Note: knxd uses --listen-tcp[=PORT] format, so we use -i=PORT or -iPORT
	if c.ListenTCP {
		args = append(args, fmt.Sprintf("-i%d", c.TCPPort))
	}

	// Group cache (-c) — routes group telegrams between clients and backend
	if c.GroupCache {
		args = append(args, "-c")
	}

	// Log level and trace flags
	if c.LogLevel > 0 {
		args = append(args, fmt.Sprintf("-f%d", c.LogLevel))
	}
	if c.TraceFlags > 0 {
		args = append(args, fmt.Sprintf("-t%d", c.TraceFlags))
	}

	// Backend
	args = append(args, "-b", c.Backend.BuildArg())

	return args
}

// BuildArg constructs the backend argument for knxd.
func (b *BackendConfig) BuildArg() string {
	switch b.Type {
	case BackendUSB:
		if b.USBDevice != "" {
			return fmt.Sprintf("usb:%s", b.USBDevice)
		}
		return "usb:"

	case BackendIPTunnel:
		port := b.Port
		if port == 0 {
			port = 3671
		}
		return fmt.Sprintf("ipt:%s:%d", b.Host, port)

	case BackendIPRouting:
		addr := b.MulticastAddress
		if addr == "" {
			addr = "224.0.23.12"
		}
		if b.Interface != "" {
			return fmt.Sprintf("ip:%s@%s", addr, b.Interface)
		}
		return fmt.Sprintf("ip:%s", addr)

	default:
		return "usb:"
	}
}

// KNX address validation pattern: area.line.device
var knxAddressPattern = regexp.MustCompile(`^\d{1,2}\.\d{1,2}\.\d{1,3}$`)

// USB ID validation pattern: 4-character hex string
var usbIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{4}$`)

// validateUSBID ensures a USB vendor/product ID is a valid 4-character hex string.
func validateUSBID(id, fieldName string) error {
	if !usbIDPattern.MatchString(id) {
		return fmt.Errorf("%s must be a 4-character hex string (e.g., 0e77)", fieldName)
	}
	return nil
}

// safePathPattern allows alphanumeric, hyphen, underscore, forward slash, and colon.
// This prevents shell metacharacters that could enable command injection.
var safePathPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-/:]+$`)

// validateSafePathComponent ensures a string doesn't contain shell metacharacters.
// This prevents command injection when the value is passed to subprocess arguments.
func validateSafePathComponent(value, fieldName string) error {
	if !safePathPattern.MatchString(value) {
		return fmt.Errorf("%s contains invalid characters (allowed: alphanumeric, hyphen, underscore, slash, colon)", fieldName)
	}
	// Additionally reject common shell metacharacters explicitly
	for _, c := range []string{";", "|", "&", "$", "`", "(", ")", "{", "}", "<", ">", "!", "\\", "'", "\""} {
		if strings.Contains(value, c) {
			return fmt.Errorf("%s contains forbidden character %q", fieldName, c)
		}
	}
	return nil
}

// Client addresses pattern: area.line.device:count
var clientAddressPattern = regexp.MustCompile(`^\d{1,2}\.\d{1,2}\.\d{1,3}:\d+$`)

func validateKNXAddress(addr string) error {
	if !knxAddressPattern.MatchString(addr) {
		return fmt.Errorf("must be in format area.line.device (e.g., 0.0.1)")
	}

	parts := strings.Split(addr, ".")
	// Regex already validated format, so Atoi won't fail, but check anyway for safety
	area, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid area: %w", err)
	}
	line, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid line: %w", err)
	}
	device, err := strconv.Atoi(parts[2])
	if err != nil {
		return fmt.Errorf("invalid device: %w", err)
	}

	if area < 0 || area > 15 {
		return fmt.Errorf("area must be 0-15")
	}
	if line < 0 || line > 15 {
		return fmt.Errorf("line must be 0-15")
	}
	if device < 0 || device > 255 {
		return fmt.Errorf("device must be 0-255")
	}

	return nil
}

func validateClientAddresses(addr string) error {
	if !clientAddressPattern.MatchString(addr) {
		return fmt.Errorf("must be in format area.line.device:count (e.g., 0.0.2:8)")
	}

	parts := strings.Split(addr, ":")
	if err := validateKNXAddress(parts[0]); err != nil {
		return err
	}

	count, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid count: %w", err)
	}
	if count < 1 || count > 255 {
		return fmt.Errorf("count must be 1-255")
	}

	return nil
}

// ConnectionURL returns the URL for connecting to knxd.
// This is used by the KNX bridge to know where to connect.
func (c *Config) ConnectionURL() string {
	if c.ListenTCP {
		return fmt.Sprintf("tcp://localhost:%d", c.TCPPort)
	}
	if c.UnixSocket != "" {
		return fmt.Sprintf("unix://%s", c.UnixSocket)
	}
	return "tcp://localhost:6720"
}

// Group address validation pattern: main/middle/sub (3-level)
var groupAddressPattern = regexp.MustCompile(`^\d{1,2}/\d{1,2}/\d{1,3}$`)

// ParseGroupAddress parses a KNX group address string (e.g., "1/7/0") into
// its 16-bit wire format representation.
//
// KNX group addresses use 3-level notation: main/middle/sub
//   - main: 0-31 (5 bits)
//   - middle: 0-7 (3 bits)
//   - sub: 0-255 (8 bits)
//
// Wire format: MMMM MSSS SSSS SSSS (big-endian uint16)
//
//	M = main (5 bits), S = middle+sub (3+8 = 11 bits)
func ParseGroupAddress(addr string) (uint16, error) {
	if !groupAddressPattern.MatchString(addr) {
		return 0, fmt.Errorf("must be in format main/middle/sub (e.g., 1/7/0)")
	}

	parts := strings.Split(addr, "/")

	main, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid main group: %w", err)
	}
	middle, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid middle group: %w", err)
	}
	sub, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, fmt.Errorf("invalid sub group: %w", err)
	}

	// Validate ranges
	if main < 0 || main > 31 {
		return 0, fmt.Errorf("main group must be 0-31")
	}
	if middle < 0 || middle > 7 {
		return 0, fmt.Errorf("middle group must be 0-7")
	}
	if sub < 0 || sub > 255 {
		return 0, fmt.Errorf("sub group must be 0-255")
	}

	// Encode to wire format: main(5) | middle(3) | sub(8)
	// Bounds validated above: main 0-31, middle 0-7, sub 0-255 all fit in uint16.
	ga := uint16(main)<<11 | uint16(middle)<<8 | uint16(sub) //nolint:gosec // G115: bounds validated above
	return ga, nil
}

// FormatGroupAddress converts a 16-bit group address to its string representation.
func FormatGroupAddress(ga uint16) string {
	main := (ga >> 11) & 0x1F
	middle := (ga >> 8) & 0x07
	sub := ga & 0xFF
	return fmt.Sprintf("%d/%d/%d", main, middle, sub)
}

// Individual address validation pattern: area.line.device
var individualAddressPattern = regexp.MustCompile(`^\d{1,2}\.\d{1,2}\.\d{1,3}$`)

// ParseIndividualAddress parses a KNX individual address string (e.g., "1.1.10") into
// its 16-bit wire format representation.
//
// KNX individual addresses use dotted notation: area.line.device
//   - area: 0-15 (4 bits)
//   - line: 0-15 (4 bits)
//   - device: 0-255 (8 bits)
//
// Wire format: AAAA LLLL DDDD DDDD (big-endian uint16)
func ParseIndividualAddress(addr string) (uint16, error) {
	if !individualAddressPattern.MatchString(addr) {
		return 0, fmt.Errorf("invalid individual address format %q (use: area.line.device)", addr)
	}

	parts := strings.Split(addr, ".")

	area, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid area: %w", err)
	}
	line, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid line: %w", err)
	}
	device, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, fmt.Errorf("invalid device: %w", err)
	}

	if area < 0 || area > 15 {
		return 0, fmt.Errorf("area must be 0-15")
	}
	if line < 0 || line > 15 {
		return 0, fmt.Errorf("line must be 0-15")
	}
	if device < 0 || device > 255 {
		return 0, fmt.Errorf("device must be 0-255")
	}

	// Encode to wire format: area(4) | line(4) | device(8)
	// Bounds validated above: area 0-15, line 0-15, device 0-255 all fit in uint16.
	ia := uint16(area)<<12 | uint16(line)<<8 | uint16(device) //nolint:gosec // G115: bounds validated above
	return ia, nil
}

// FormatIndividualAddress converts a 16-bit individual address to its string representation.
func FormatIndividualAddress(ia uint16) string {
	area := (ia >> 12) & 0x0F
	line := (ia >> 8) & 0x0F
	device := ia & 0xFF
	return fmt.Sprintf("%d.%d.%d", area, line, device)
}
