package model

// DeviceGroup represents a device group configuration
type DeviceGroup struct {
	Name         string
	ObjectGroups []string
}

// NewDeviceGroup creates a new device group
func NewDeviceGroup(name string) *DeviceGroup {
	return &DeviceGroup{
		Name:         name,
		ObjectGroups: []string{},
	}
}
