// Package model contains the data models for the application
package model

// Device represents an SNMP device configuration
type Device struct {
	Name                string
	IP                  string
	Port                int
	PollInterval        int
	Timeout             int
	Retries             int
	ExponentialTimeout  bool
	Version             string // "2c" or "3"
	Communities         []string
	V3Credentials       []V3Credential
	DeviceGroups        []string
	MaxConcurrentPolls  int
	SourceFile          string  // Added to store the source YAML file path
}

// V3Credential represents SNMPv3 authentication credentials
type V3Credential struct {
	Username                string
	AuthenticationProtocol  string
	AuthenticationPassphrase string
	PrivacyProtocol         string
	PrivacyPassphrase       string
}

// NewDevice creates a new device with default values
func NewDevice(name string) *Device {
	return &Device{
		Name:                name,
		Port:                161,
		PollInterval:        60,
		Timeout:             5000,
		Retries:             2,
		ExponentialTimeout:  false,
		Version:             "2c",
		Communities:         []string{"public"},
		DeviceGroups:        []string{},
		MaxConcurrentPolls:  1,
		SourceFile:          "",  // Default source file is empty
	}
}
