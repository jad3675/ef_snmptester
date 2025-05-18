package model

import "time"

// TestType represents the type of test
type TestType string

const (
	// ConnectivityTest is a test for SNMP connectivity
	ConnectivityTest TestType = "connectivity"
	
	// GroupTest is a test for a device group
	GroupTest TestType = "group"
)

// Status represents the status of a test
type Status string

const (
	// Success indicates a successful test
	Success Status = "success"
	
	// Failure indicates a failed test
	Failure Status = "failure"
	
	// Pending indicates a pending test
	Pending Status = "pending"
	
	// Running indicates a running test
	Running Status = "running"
)

// TestResult represents the result of a test
type TestResult struct {
	DeviceName string
	DeviceIP   string
	SourceFile string  // Added to store the source YAML file
	TestType   TestType
	Status     Status
	Message    string
	Error      error
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Data       map[string]string                 // Additional result data
	WalkedOidData map[string]map[string]interface{} // For detailed SNMP walk results
}

// NewTestResult creates a new test result
func NewTestResult(deviceName, deviceIP, sourceFile string, testType TestType) *TestResult {
	return &TestResult{
		DeviceName: deviceName,
		DeviceIP:   deviceIP,
		SourceFile: sourceFile,
		TestType:   testType,
		Status:     Pending,
		StartTime:  time.Now(),
		Data:       make(map[string]string),
		WalkedOidData: make(map[string]map[string]interface{}),
	}
}

// Complete marks the test as complete
func (r *TestResult) Complete(status Status, message string, err error) {
	r.Status = status
	r.Message = message
	r.Error = err
	r.EndTime = time.Now()
	r.Duration = r.EndTime.Sub(r.StartTime)
}

// Success marks the test as successful
func (r *TestResult) Success(message string) {
	r.Complete(Success, message, nil)
}

// Failure marks the test as failed
func (r *TestResult) Failure(message string, err error) {
	r.Complete(Failure, message, err)
}
