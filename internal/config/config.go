// Package config provides functionality for loading and parsing ElastiFlow configuration
package config

import (
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/jad3675/ef_snmptester/internal/model"
	"gopkg.in/yaml.v3"
)

// Manager manages loading and accessing ElastiFlow SNMP configuration
type Manager struct {
	BasePath     string
	DevicesPath  string
	GroupsPath   string
	ObjGroupsPath string
	ObjectsPath  string
	
	Devices      map[string]*model.Device
	DeviceGroups map[string]*model.DeviceGroup
	ObjectGroups map[string]*model.ObjectGroup
	Objects      map[string]*model.Object
}

// NewManager creates a new configuration manager
func NewManager(basePath string) *Manager {
	return &Manager{
		BasePath:     basePath,
		DevicesPath:  filepath.Join(basePath, "devices"),
		GroupsPath:   filepath.Join(basePath, "device_groups"),
		ObjGroupsPath: filepath.Join(basePath, "object_groups"),
		ObjectsPath:  filepath.Join(basePath, "objects"),
		
		Devices:      make(map[string]*model.Device),
		DeviceGroups: make(map[string]*model.DeviceGroup),
		ObjectGroups: make(map[string]*model.ObjectGroup),
		Objects:      make(map[string]*model.Object),
	}
}

// LoadAll loads all configuration components
func (m *Manager) LoadAll() error {
	if err := m.LoadDevices(); err != nil {
		return fmt.Errorf("failed to load devices: %w", err)
	}
	
	if err := m.LoadDeviceGroups(); err != nil {
		return fmt.Errorf("failed to load device groups: %w", err)
	}
	
	if err := m.LoadObjectGroups(); err != nil {
		return fmt.Errorf("failed to load object groups: %w", err)
	}
	
	if err := m.LoadObjects(); err != nil {
		return fmt.Errorf("failed to load objects: %w", err)
	}
	
	return nil
}

// LoadDevices loads all device configurations
func (m *Manager) LoadDevices() error {
	files, err := filepath.Glob(filepath.Join(m.DevicesPath, "*.yml"))
	if err != nil {
		return err
	}
	
	for _, file := range files {
		if err := m.loadDeviceFile(file); err != nil {
			return fmt.Errorf("error loading device file %s: %w", file, err)
		}
	}
	
	return nil
}

// loadDeviceFile loads devices from a single YAML file
func (m *Manager) loadDeviceFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	
	var deviceMap map[string]interface{}
	if err := yaml.Unmarshal(data, &deviceMap); err != nil {
		return err
	}
	
	// Get just the filename from the path for display purposes
	sourceFile := filepath.Base(filePath)
	
	for deviceName, deviceData := range deviceMap {
		device := model.NewDevice(deviceName)
		device.SourceFile = sourceFile // Store the source file
		
		deviceMap, ok := deviceData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid device data format for %s", deviceName)
		}
		
		// Parse device properties
		if ip, ok := deviceMap["ip"].(string); ok {
			device.IP = ip
		}
		
		if port, ok := deviceMap["port"].(int); ok {
			device.Port = port
		}
		
		if pollInterval, ok := deviceMap["poll_interval"].(int); ok {
			device.PollInterval = pollInterval
		}
		
		if timeout, ok := deviceMap["timeout"].(int); ok {
			device.Timeout = timeout
		}
		
		if retries, ok := deviceMap["retries"].(int); ok {
			device.Retries = retries
		}
		
		if expTimeout, ok := deviceMap["exponential_timeout"].(bool); ok {
			device.ExponentialTimeout = expTimeout
		}
		
		if versionVal, ok := deviceMap["version"]; ok { // Get the value first. Using deviceMap as it's the correct variable in this scope.
			switch v := versionVal.(type) {
			case string:
				device.Version = v
			case int:
				device.Version = fmt.Sprintf("%d", v) // Convert int to string
			default:
				// Optionally handle or log an unexpected type for version
				// For now, we'll let it keep the default if type is neither string nor int
			}
		}
		
		// Parse communities (for SNMPv2c)
		if communities, ok := deviceMap["communities"].([]interface{}); ok {
			device.Communities = make([]string, 0, len(communities))
			for _, community := range communities {
				if communityStr, ok := community.(string); ok {
					device.Communities = append(device.Communities, communityStr)
				}
			}
		}
		
		// Parse v3 credentials (for SNMPv3)
		if v3Creds, ok := deviceMap["v3_credentials"].([]interface{}); ok {
			device.V3Credentials = make([]model.V3Credential, 0, len(v3Creds))
			for _, v3Cred := range v3Creds {
				if credMap, ok := v3Cred.(map[string]interface{}); ok {
					cred := model.V3Credential{}
					
					if username, ok := credMap["username"].(string); ok {
						cred.Username = username
					}
					
					if authProto, ok := credMap["authentication_protocol"].(string); ok {
						cred.AuthenticationProtocol = authProto
					}
					
					if authPass, ok := credMap["authentication_passphrase"].(string); ok {
						cred.AuthenticationPassphrase = authPass
					}
					
					if privProto, ok := credMap["privacy_protocol"].(string); ok {
						cred.PrivacyProtocol = privProto
					}
					
					if privPass, ok := credMap["privacy_passphrase"].(string); ok {
						cred.PrivacyPassphrase = privPass
					}
					
					device.V3Credentials = append(device.V3Credentials, cred)
				}
			}
		}
		
		// Parse device groups
		if groups, ok := deviceMap["device_groups"].([]interface{}); ok {
			device.DeviceGroups = make([]string, 0, len(groups))
			for _, group := range groups {
				if groupStr, ok := group.(string); ok {
					device.DeviceGroups = append(device.DeviceGroups, groupStr)
				}
			}
		}
		
		// Parse max concurrent polls
		if maxPolls, ok := deviceMap["max_concurrent_polls"].(int); ok {
			device.MaxConcurrentPolls = maxPolls
		}
		
		m.Devices[deviceName] = device
	}
	
	return nil
}

// LoadDeviceGroups loads all device group configurations
func (m *Manager) LoadDeviceGroups() error {
	files, err := filepath.Glob(filepath.Join(m.GroupsPath, "*.yml"))
	if err != nil {
		return err
	}
	
	for _, file := range files {
		if err := m.loadDeviceGroupFile(file); err != nil {
			return fmt.Errorf("error loading device group file %s: %w", file, err)
		}
	}
	
	return nil
}

// loadDeviceGroupFile loads device groups from a single YAML file
func (m *Manager) loadDeviceGroupFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	
	var groupMap map[string]interface{}
	if err := yaml.Unmarshal(data, &groupMap); err != nil {
		return err
	}
	
	for groupName, groupData := range groupMap {
		group := model.NewDeviceGroup(groupName)
		
		groupMap, ok := groupData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid device group data format for %s", groupName)
		}
		
		// Parse object groups
		if objGroups, ok := groupMap["object_groups"].([]interface{}); ok {
			group.ObjectGroups = make([]string, 0, len(objGroups))
			for _, objGroup := range objGroups {
				if objGroupStr, ok := objGroup.(string); ok {
					group.ObjectGroups = append(group.ObjectGroups, objGroupStr)
				}
			}
		}
		
		m.DeviceGroups[groupName] = group
	}
	
	return nil
}

// LoadObjectGroups loads all object group configurations
func (m *Manager) LoadObjectGroups() error {
	// First load from the root object_groups directory
	files, err := filepath.Glob(filepath.Join(m.ObjGroupsPath, "*.yml"))
	if err != nil {
		return err
	}
	
	for _, file := range files {
		if err := m.loadObjectGroupFile(file); err != nil {
			return fmt.Errorf("error loading object group file %s: %w", file, err)
		}
	}
	
	// Then load from subdirectories (like "ietf", "cisco", etc.)
	dirs, err := os.ReadDir(m.ObjGroupsPath)
	if err != nil {
		return err
	}
	
	for _, dir := range dirs {
		if dir.IsDir() {
			subdir := filepath.Join(m.ObjGroupsPath, dir.Name())
			subfiles, err := filepath.Glob(filepath.Join(subdir, "*.yml"))
			if err != nil {
				return err
			}
			
			for _, file := range subfiles {
				if err := m.loadObjectGroupFile(file); err != nil {
					return fmt.Errorf("error loading object group file %s: %w", file, err)
				}
			}
		}
	}
	
	return nil
}

// loadObjectGroupFile loads object groups from a single YAML file
func (m *Manager) loadObjectGroupFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	
	var objGroupMap map[string]interface{}
	if err := yaml.Unmarshal(data, &objGroupMap); err != nil {
		return err
	}
	
	for objGroupName, objGroupData := range objGroupMap {
		objGroup := model.NewObjectGroup(objGroupName)
		
		objGroupMap, ok := objGroupData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid object group data format for %s", objGroupName)
		}
		
		// Parse objects
		if objects, ok := objGroupMap["objects"].([]interface{}); ok {
			objGroup.Objects = make([]string, 0, len(objects))
			for _, object := range objects {
				if objectStr, ok := object.(string); ok {
					objGroup.Objects = append(objGroup.Objects, objectStr)
				}
			}
		}
		
		m.ObjectGroups[objGroupName] = objGroup
	}
	
	return nil
}

// LoadObjects loads all object configurations
func (m *Manager) LoadObjects() error {
	// We need to handle subdirectories as objects are organized by vendor/standard
	dirs, err := os.ReadDir(m.ObjectsPath)
	if err != nil {
		return err
	}
	
	for _, dir := range dirs {
		if dir.IsDir() {
			subdir := filepath.Join(m.ObjectsPath, dir.Name())
			subfiles, err := filepath.Glob(filepath.Join(subdir, "*.yml"))
			if err != nil {
				continue
			}
			
			for _, file := range subfiles {
				if err := m.loadObjectFile(file); err != nil {
					return fmt.Errorf("error loading object file %s: %w", file, err)
				}
			}
		}
	}
	
	return nil
}

// loadObjectFile loads objects from a single YAML file
func (m *Manager) loadObjectFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	
	var objMap map[string]interface{}
	if err := yaml.Unmarshal(data, &objMap); err != nil {
		return err
	}
	
	for objName, objData := range objMap {
		obj := model.NewObject(objName)
		
		objMap, ok := objData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid object data format for %s", objName)
		}
		
		// Parse MIB
		if mib, ok := objMap["mib"].(string); ok {
			obj.MIB = mib
		}
		
		// Parse discovery attribute
		if discoveryAttr, ok := objMap["discovery_attribute"].(string); ok {
			obj.DiscoveryAttribute = discoveryAttr
		}
		
		// Parse attributes
		if attrs, ok := objMap["attributes"].(map[string]interface{}); ok {
			for attrName, attrData := range attrs {
				attr := model.Attribute{}
				
				attrMap, ok := attrData.(map[string]interface{})
				if !ok {
					continue
				}
				
				if oid, ok := attrMap["oid"].(string); ok {
					attr.OID = oid
				}
				
				if name, ok := attrMap["name"].(string); ok {
					attr.Name = name
				}
				
				if syntax, ok := attrMap["syntax"].(string); ok {
					attr.Syntax = syntax
				}
				
				if rediscover, ok := attrMap["rediscover"].(string); ok {
					attr.Rediscover = rediscover
				}
				
				obj.Attributes[attrName] = attr
			}
		}
		
		m.Objects[objName] = obj
	}
	
	return nil
}

// GetDeviceOIDs returns all OIDs for a specific device group
func (m *Manager) GetDeviceOIDs(deviceName string, groupName string) (map[string]string, error) {
	device, ok := m.Devices[deviceName]
	if !ok {
		return nil, fmt.Errorf("device %s not found", deviceName)
	}
	
	deviceGroup, ok := m.DeviceGroups[groupName]
	if !ok {
		return nil, fmt.Errorf("device group %s not found", groupName)
	}
	
	// Check if the device belongs to the group
	belongsToGroup := false
	for _, dg := range device.DeviceGroups {
		if dg == groupName {
			belongsToGroup = true
			break
		}
	}
	
	if !belongsToGroup {
		return nil, fmt.Errorf("device %s does not belong to group %s", deviceName, groupName)
	}
	
	// Collect all OIDs from object groups
	oids := make(map[string]string)
	for _, objGroupName := range deviceGroup.ObjectGroups {
		objGroup, ok := m.ObjectGroups[objGroupName]
		if !ok {
			continue
		}
		
		for _, objName := range objGroup.Objects {
			obj, ok := m.Objects[objName]
			if !ok {
				continue
			}
			
			for attrName, attr := range obj.Attributes {
				oidName := fmt.Sprintf("%s.%s", objName, attrName)
				oids[oidName] = attr.OID
			}
		}
	}
	
	return oids, nil
}

// GetAllDeviceGroups returns all device groups for a device
func (m *Manager) GetAllDeviceGroups(deviceName string) ([]string, error) {
	device, ok := m.Devices[deviceName]
	if !ok {
		return nil, fmt.Errorf("device %s not found", deviceName)
	}
	
	return device.DeviceGroups, nil
}
