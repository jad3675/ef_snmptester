// Package snmp provides functionality for SNMP operations
package snmp

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jad3675/ef_snmptester/internal/model"
	"github.com/gosnmp/gosnmp"
)

// Client represents an SNMP client
type Client struct {
	Device *model.Device
	client *gosnmp.GoSNMP
}

// NewClient creates a new SNMP client for a device
func NewClient(device *model.Device) (*Client, error) {
	var client *gosnmp.GoSNMP

	switch device.Version {
	case "2c":
		if len(device.Communities) == 0 {
			return nil, fmt.Errorf("no communities defined for SNMPv2c device %s", device.Name)
		}
		
		client = &gosnmp.GoSNMP{
			Target:    device.IP,
			Port:      uint16(device.Port),
			Community: device.Communities[0],
			Version:   gosnmp.Version2c,
			Timeout:   time.Duration(device.Timeout) * time.Millisecond,
			Retries:   device.Retries,
		}
		
	case "3":
		if len(device.V3Credentials) == 0 {
			return nil, fmt.Errorf("no v3 credentials defined for SNMPv3 device %s", device.Name)
		}
		
		v3Cred := device.V3Credentials[0]
		
		// Map authentication protocol (case-insensitive)
		var authProto gosnmp.SnmpV3AuthProtocol
		switch strings.ToLower(v3Cred.AuthenticationProtocol) {
		case "sha", "sha1":
			authProto = gosnmp.SHA
		case "md5":
			authProto = gosnmp.MD5
		default:
			return nil, fmt.Errorf("unsupported authentication protocol: %s", v3Cred.AuthenticationProtocol)
		}
		
		// Map privacy protocol (case-insensitive)
		var privProto gosnmp.SnmpV3PrivProtocol
		switch strings.ToLower(v3Cred.PrivacyProtocol) {
		case "des":
			privProto = gosnmp.DES
		case "aes", "aes128":
			privProto = gosnmp.AES
		default:
			return nil, fmt.Errorf("unsupported privacy protocol: %s", v3Cred.PrivacyProtocol)
		}
		
		// Create the SNMPv3 client with security parameters
		client = &gosnmp.GoSNMP{
			Target:    device.IP,
			Port:      uint16(device.Port),
			Version:   gosnmp.Version3,
			Timeout:   time.Duration(device.Timeout) * time.Millisecond,
			Retries:   device.Retries,
			SecurityParameters: &gosnmp.UsmSecurityParameters{
				UserName:                 v3Cred.Username,
				AuthenticationProtocol:   authProto,
				AuthenticationPassphrase: v3Cred.AuthenticationPassphrase,
				PrivacyProtocol:          privProto,
				PrivacyPassphrase:        v3Cred.PrivacyPassphrase,
			},
			SecurityModel: gosnmp.UserSecurityModel, // Explicitly set the security model
			MsgFlags:      gosnmp.AuthPriv,         // Set security level to AuthPriv (authentication + privacy)
		}

	default:
		return nil, fmt.Errorf("unsupported SNMP version: %s", device.Version)
	}
	
	return &Client{
		Device: device,
		client: client,
	}, nil
}

// Connect establishes a connection to the SNMP device
func (c *Client) Connect() error {
	return c.client.Connect()
}

// Close closes the SNMP connection
func (c *Client) Close() error {
	return c.client.Conn.Close()
}

// TestConnectivity tests basic SNMP connectivity by getting sysObjectID
func (c *Client) TestConnectivity() (*model.TestResult, error) {
	result := model.NewTestResult(c.Device.Name, c.Device.IP, c.Device.SourceFile, model.ConnectivityTest)
	result.Status = model.Running
	
	// Try to connect to the device
	if err := c.Connect(); err != nil {
		result.Failure(fmt.Sprintf("Connection failed: %s", err), err)
		return result, err
	}
	defer c.Close()
	
	// Get the sysObjectID
	oid := ".1.3.6.1.2.1.1.2.0" // sysObjectID
	pdu, err := c.client.Get([]string{oid})
	if err != nil {
		result.Failure(fmt.Sprintf("Failed to get sysObjectID: %s", err), err)
		return result, err
	}
	
	// Check if we got a valid response
	foundSysObjectID := false
	var sysObjectIDValue string
	
	for _, variable := range pdu.Variables {
		if variable.Name == oid {
			foundSysObjectID = true
			
			// Format the sysObjectID value based on type
			switch variable.Type {
			case gosnmp.ObjectIdentifier:
				if oidVal, ok := variable.Value.(string); ok { // Renamed oid to oidVal to avoid conflict
					sysObjectIDValue = oidVal
				} else {
					sysObjectIDValue = fmt.Sprintf("%v", variable.Value)
				}
			default:
				sysObjectIDValue = fmt.Sprintf("%v", variable.Value)
			}
			
			result.Data["sysObjectID"] = sysObjectIDValue
			break
		}
	}
	
	if !foundSysObjectID {
		result.Failure("Failed to get sysObjectID: OID not found in response", 
			fmt.Errorf("OID not found in response"))
		return result, fmt.Errorf("OID not found in response")
	}
	
	// Mark as successful and store sysObjectID in the message
	result.Success(fmt.Sprintf("sysObjectID: %s", sysObjectIDValue))
	return result, nil
}

// walkOIDParallel performs an SNMP walk for a single OID
func (c *Client) walkOIDParallel(oidName, oid string, wg *sync.WaitGroup, results map[string]map[string]interface{}, resultsMutex *sync.Mutex, failureCountChan chan<- bool) {
	defer wg.Done()
	
	// Initialize result for this OID
	localResults := make(map[string]interface{})
	
	// Create a new client specific to this goroutine to avoid concurrency issues
	// with a shared GoSNMP instance
	newClient, err := NewClient(c.Device)
	if err != nil {
		localResults["error"] = fmt.Sprintf("Failed to create SNMP client: %s", err)
		// Update shared results with a mutex to avoid race conditions
		resultsMutex.Lock()
		results[oidName] = localResults
		resultsMutex.Unlock()
		failureCountChan <- true // Signal a failure
		return
	}
	
	// Connect to the device
	if err := newClient.Connect(); err != nil {
		localResults["error"] = fmt.Sprintf("Connection failed: %s", err)
		resultsMutex.Lock()
		results[oidName] = localResults
		resultsMutex.Unlock()
		failureCountChan <- true // Signal a failure
		return
	}
	defer newClient.Close()
	
	// Walk the OID
	err = newClient.client.Walk(oid, func(pdu gosnmp.SnmpPDU) error {
		// Extract the instance part from the OID
		instance := pdu.Name[len(oid):]
		if instance == "" {
			instance = ".0" // For scalar objects
		}
		
		// Store the value with the instance as the key
		localResults[instance] = pdu.Value
		return nil
	})
	
	if err != nil {
		localResults["error"] = err.Error()
		failureCountChan <- true // Signal a failure
	} else if len(localResults) == 0 {
		// No error but no data either - not considered a failure
		failureCountChan <- false // No failure
	} else {
		// Success with data
		failureCountChan <- false // No failure
	}
	
	// Update the shared results with a mutex
	resultsMutex.Lock()
	results[oidName] = localResults
	resultsMutex.Unlock()
}

// WalkOIDs performs SNMP walk operations for a list of OIDs in parallel with early failure detection
func (c *Client) WalkOIDs(oids map[string]string) (map[string]map[string]interface{}, error) {
	results := make(map[string]map[string]interface{})
	resultsMutex := &sync.Mutex{} // Mutex for safe concurrent access to results map
	
	// Determine the maximum concurrent operations based on device config
	maxConcurrent := c.Device.MaxConcurrentPolls
	if maxConcurrent <= 0 {
		// Default to a reasonable number if not specified
		maxConcurrent = 5
	}
	
	// Use a semaphore pattern to limit concurrency
	sem := make(chan struct{}, maxConcurrent)
	
	// WaitGroup to track completion of all goroutines
	var wg sync.WaitGroup
	
	// Channel to track failure count for early abort
	failureCountChan := make(chan bool, len(oids))
	
	// Track OIDs for early abort detection
	var consecutiveFailures int
	var processedOIDs = 0
	var skipRemaining = false
	
	// Process OIDs
	for oidName, oid := range oids {
		// Skip remaining OIDs if we've hit the failure threshold
		if skipRemaining {
			// Mark as skipped without testing
			resultsMutex.Lock()
			results[oidName] = map[string]interface{}{
				"error": "Early abort: device appears to be unreachable",
			}
			resultsMutex.Unlock()
			continue
		}
		
		processedOIDs++
		
		// Acquire semaphore slot (blocks if maxConcurrent goroutines are already running)
		sem <- struct{}{}
		
		// Create waitgroup entry
		wg.Add(1)
		
		// Start goroutine for this OID
		go func(oidName, oid string) {
			defer func() { <-sem }() // Release semaphore slot when done
			
			c.walkOIDParallel(oidName, oid, &wg, results, resultsMutex, failureCountChan)
		}(oidName, oid)
		
		// Check failure count after each batch completes
		// This helps detect consistent failures early without waiting for all OIDs
		if processedOIDs%maxConcurrent == 0 || processedOIDs == len(oids) {
			// Wait for the current batch to complete
			for i := 0; i < maxConcurrent && i < len(oids); i++ {
				select {
				case isFailure := <-failureCountChan:
					if isFailure {
						consecutiveFailures++
					} else {
						consecutiveFailures = 0
					}
				default:
					// No result yet, continue
				}
			}
			
			// Check if we should abort
			if consecutiveFailures >= 5 {
				skipRemaining = true
				// Calculate remaining OIDs
				// remainingOIDs := totalOIDs - processedOIDs // This calculation is not used after removing Printf
			}
		}
	}
	
	// Wait for all goroutines to complete
	wg.Wait()
	
	return results, nil
}

// TestGroupOIDs tests a set of OIDs for a device group
func (c *Client) TestGroupOIDs(oids map[string]string) (*model.TestResult, error) {
	result := model.NewTestResult(c.Device.Name, c.Device.IP, c.Device.SourceFile, model.GroupTest)
	result.Status = model.Running
	
	walkResults, err := c.WalkOIDs(oids)
	if err != nil {
		result.Failure("Failed to walk OIDs", err)
		return result, err
	}
	
	// Count successful OIDs
	successful := 0
	for _, results := range walkResults {
		if _, ok := results["error"]; !ok && len(results) > 0 {
			successful++
		}
	}
	
	if successful == 0 {
		result.Failure(fmt.Sprintf("Failed to retrieve any of the %d OIDs", len(oids)), nil)
	} else if successful < len(oids) {
		result.Success(fmt.Sprintf("Successfully retrieved %d of %d OIDs", successful, len(oids)))
	} else {
		result.Success(fmt.Sprintf("Successfully retrieved all %d OIDs", len(oids)))
	}
	
	// Store summary in result data
	result.Data["total_oids"] = fmt.Sprintf("%d", len(oids))
	result.Data["successful_oids"] = fmt.Sprintf("%d", successful)
	result.Data["failed_oids"] = fmt.Sprintf("%d", len(oids)-successful)
	result.WalkedOidData = walkResults // Store the detailed walk results
	
	return result, nil
}
