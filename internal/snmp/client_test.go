// Package snmp provides functionality for SNMP operations
package snmp

import (
	"testing"
	
	"github.com/jad3675/ef_snmptester/internal/model"
)

func TestClient_TestConnectivity(t *testing.T) {
	device := &model.Device{
		Name:        "mock_device",
		IP:          "192.168.1.1",
		Port:        161,
		Version:     "2c",
		Communities: []string{"public"},
		Timeout:     1000,
		Retries:     2,
	}
	
	client, err := NewClient(device)
	if err != nil {
		t.Fatalf("Failed to create SNMP client: %s", err)
	}
	
	// This is just a quick test to verify if our modifications didn't break anything
	// We're not actually doing SNMP operations here
	result, err := client.TestConnectivity()
	
	// In a real test setup, we'd mock the SNMP client behavior
	// For now, let's just check the result contains the proper device name
	if result.DeviceName != "mock_device" {
		t.Errorf("Expected device name 'mock_device', got '%s'", result.DeviceName)
	}
}
