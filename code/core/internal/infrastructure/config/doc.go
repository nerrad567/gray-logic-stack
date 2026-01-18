// Package config handles loading and validating Gray Logic Core configuration.
//
// This package manages:
//   - Loading configuration from YAML files
//   - Overriding with environment variables
//   - Validation of required fields
//   - Default value handling
//
// Security Considerations:
//   - Sensitive values (passwords, tokens) should be set via environment variables
//   - The config file should have restricted permissions (0600)
//   - JWT secrets must be changed from defaults before production use
//
// Performance Characteristics:
//   - Configuration is loaded once at startup
//   - No runtime overhead after initial load
//
// Usage:
//
//	cfg, err := config.Load("configs/config.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(cfg.Site.Name)
//
// Related Documents:
//   - docs/operations/bootstrapping.md — System initialisation procedures
//   - docs/architecture/security-model.md — Security architecture
package config
