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

// TestGroupOIDs tests a set of OID Names for a device group by attempting to GetNext on each.
func (c *Client) TestGroupOIDs(oidNameMap map[string]string) (*model.TestResult, error) {
	result := model.NewTestResult(c.Device.Name, c.Device.IP, c.Device.SourceFile, model.GroupTest)
	result.Status = model.Running

	// Use GetFirstEntryForOidNames instead of WalkOIDs
	retrievedOidData, err := c.GetFirstEntryForOidNames(oidNameMap)
	if err != nil {
		// This error from GetFirstEntryForOidNames is currently always nil,
		// but good practice to check.
		result.Failure("Failed to perform GetNext operations for OID names", err)
		return result, err
	}

	// Count successful GetNext operations (i.e., OID names that returned a value)
	successfulOidNameCount := 0
	for _, dataMap := range retrievedOidData {
		// A successful GetNext for an OID Name means no "error" key in its specific map.
		if _, hasError := dataMap["error"]; !hasError {
			// And it should have retrieved a value.
			if _, hasValue := dataMap["value_raw"]; hasValue {
				successfulOidNameCount++
			}
		}
	}

	totalOidNames := len(oidNameMap)
	if successfulOidNameCount == 0 {
		result.Failure(fmt.Sprintf("Failed to retrieve any first entry for the %d OID names", totalOidNames), nil)
	} else if successfulOidNameCount < totalOidNames {
		result.Success(fmt.Sprintf("Successfully retrieved first entry for %d of %d OID names", successfulOidNameCount, totalOidNames))
	} else {
		result.Success(fmt.Sprintf("Successfully retrieved first entry for all %d OID names", totalOidNames))
	}

	// Store summary in result data
	result.Data["total_oids"] = fmt.Sprintf("%d", totalOidNames) // Represents total OID Names
	result.Data["successful_oids"] = fmt.Sprintf("%d", successfulOidNameCount) // Represents successful OID Name GetNexts
	result.Data["failed_oids"] = fmt.Sprintf("%d", totalOidNames-successfulOidNameCount)
	result.WalkedOidData = retrievedOidData // Store the detailed GetNext results

	return result, nil
}

// getFirstEntryParallel performs an SNMP GetNext for a single OID.
// It's designed to be run in a goroutine.
func (c *Client) getFirstEntryParallel(oidName, oid string, wg *sync.WaitGroup, results map[string]map[string]interface{}, resultsMutex *sync.Mutex, failureChan chan<- bool) {
	defer wg.Done()

	localResult := make(map[string]interface{})

	// Create a new client specific to this goroutine
	newClient, err := NewClient(c.Device)
	if err != nil {
		localResult["error"] = fmt.Sprintf("Failed to create SNMP client: %s", err)
		resultsMutex.Lock()
		results[oidName] = localResult
		resultsMutex.Unlock()
		failureChan <- true
		return
	}

	if err := newClient.Connect(); err != nil {
		localResult["error"] = fmt.Sprintf("Connection failed: %s", err)
		resultsMutex.Lock()
		results[oidName] = localResult
		resultsMutex.Unlock()
		failureChan <- true
		return
	}
	defer newClient.Close()

	// Perform GetNext operation for the given OID
	// We expect only one variable in the response for a GetNext on a base OID.
	pdu, err := newClient.client.GetNext([]string{oid})
	if err != nil {
		localResult["error"] = err.Error()
		failureChan <- true
	} else if pdu == nil || len(pdu.Variables) == 0 {
		localResult["error"] = "No response or empty PDU received"
		failureChan <- true
	} else if pdu.Variables[0].Type == gosnmp.NoSuchObject || pdu.Variables[0].Type == gosnmp.NoSuchInstance || pdu.Variables[0].Type == gosnmp.EndOfMibView {
		localResult["error"] = fmt.Sprintf("OID not found or end of MIB view: %s", pdu.Variables[0].Type.String())
		// This is a valid SNMP response indicating the OID doesn't exist as specified,
		// but for our purpose of "does this MIB respond", it's a failure to get a value.
		failureChan <- true
	} else {
		// Successfully retrieved an entry
		retrievedPDU := pdu.Variables[0]
		localResult["retrieved_oid"] = retrievedPDU.Name
		localResult["type"] = retrievedPDU.Type.String()
		localResult["value"] = gosnmp.ToBigInt(retrievedPDU.Value) // Convert to a common format if possible, or keep as is
		// For simplicity, store raw value. Display formatting can handle type.
		localResult["value_raw"] = retrievedPDU.Value
		failureChan <- false // Success
	}

	resultsMutex.Lock()
	results[oidName] = localResult
	resultsMutex.Unlock()
}

// GetFirstEntryForOidNames performs SNMP GetNext operations for a list of OID names in parallel.
// This is used to quickly check if a MIB/OID prefix is responsive by fetching the first entry.
func (c *Client) GetFirstEntryForOidNames(oids map[string]string) (map[string]map[string]interface{}, error) {
	results := make(map[string]map[string]interface{})
	resultsMutex := &sync.Mutex{}

	maxConcurrent := c.Device.MaxConcurrentPolls
	if maxConcurrent <= 0 {
		maxConcurrent = 5 // Default concurrency
	}
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	failureChan := make(chan bool, len(oids)) // To track if individual GetNext operations fail
	processedOidsInBatch := 0
	
	// Re-introduce early abort logic
	var consecutiveFailures int
	var skipRemaining = false
	var processedOIDsTotal = 0


	for oidName, oid := range oids {
		processedOIDsTotal++
		if skipRemaining {
			resultsMutex.Lock()
			results[oidName] = map[string]interface{}{
				"error": "Early abort: device appears to be unreachable or unresponsive to this group's OIDs",
			}
			resultsMutex.Unlock()
			// We still need to send a value to failureChan to unblock the batch processing logic later,
			// or adjust the batch processing logic. For simplicity, let's assume these don't count towards batch processing.
			// Or, more simply, just continue and don't launch a goroutine.
			// The wg.Wait() will still wait for actual launched goroutines.
			continue
		}

		sem <- struct{}{}
		wg.Add(1)
		processedOidsInBatch++
		go func(name, id string) {
			defer func() { <-sem }()
			c.getFirstEntryParallel(name, id, &wg, results, resultsMutex, failureChan)
		}(oidName, oid)

		// Check failure count after each batch completes or at the end
		// This logic is similar to the one in WalkOIDs
		if processedOidsInBatch == maxConcurrent || processedOIDsTotal == len(oids) {
			// Wait for the current batch of launched goroutines to report to failureChan
			for i := 0; i < processedOidsInBatch; i++ {
				isFailure := <-failureChan
				if isFailure {
					consecutiveFailures++
				} else {
					consecutiveFailures = 0 // Reset on any success
				}
			}
			processedOidsInBatch = 0 // Reset for next batch

			if consecutiveFailures >= 5 { // Threshold for aborting
				skipRemaining = true
			}
		}
	}

	wg.Wait()
	// Ensure any remaining failureChan messages from the last partial batch are drained
	// if the loop finished before a full batch was processed.
	// This happens if total OIDs is not a multiple of maxConcurrent.
	for i := 0; i < processedOidsInBatch; i++ {
		isFailure := <-failureChan
		if isFailure {
			consecutiveFailures++
		} else {
			consecutiveFailures = 0
		}
		// No need to check skipRemaining here again as all goroutines have been launched or skipped.
	}
	close(failureChan)


	// Check overall status - not strictly necessary to return an error here
	// as individual errors are in the results map.
	// For now, return nil error, and let caller inspect individual results.
	return results, nil
}
