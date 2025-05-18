package ui

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"
	
	"github.com/jad3675/ef_snmptester/internal/model"
	"github.com/rivo/tview"
)

// ExportConnectivityResultsToCSV exports connectivity test results to a CSV file
func ExportConnectivityResultsToCSV(results []*model.TestResult, filePath string, app *tview.Application, statusText *tview.TextView) {
	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		app.QueueUpdateDraw(func() {
			statusText.SetText(fmt.Sprintf("[red]Failed to create CSV file: %s", err))
		})
		return
	}
	defer file.Close()
	
	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Write header
	header := []string{
		"Device Name",
		"IP Address",
		"Status",
		"Message",
		"Start Time",
		"End Time",
		"Duration (ms)",
		"System Object ID",
		"Source File", // Moved to the end
	}
	if err := writer.Write(header); err != nil {
		app.QueueUpdateDraw(func() {
			statusText.SetText(fmt.Sprintf("[red]Failed to write CSV header: %s", err))
		})
		return
	}
	
	// Write data rows
	for _, result := range results {
		// Skip results that are still pending
		if result.Status == model.Pending || result.Status == model.Running {
			continue
		}
		
		// Format duration in milliseconds
		durationMs := result.Duration.Milliseconds()
		
		// Get system object ID if available
		sysObjectID := ""
		if result.Status == model.Success {
			if id, ok := result.Data["sysObjectID"]; ok {
				sysObjectID = id
			}
		}
		
		// Format message
		message := result.Message
		if result.Error != nil {
			message = result.Error.Error()
		}
		
		// Build row
		row := []string{
			result.DeviceName,
			result.DeviceIP,
			string(result.Status),
			message,
			result.StartTime.Format(time.RFC3339),
			result.EndTime.Format(time.RFC3339),
			fmt.Sprintf("%d", durationMs),
			sysObjectID,
			result.SourceFile, // Moved to the end
		}
		
		if err := writer.Write(row); err != nil {
			app.QueueUpdateDraw(func() {
				statusText.SetText(fmt.Sprintf("[red]Failed to write CSV row: %s", err))
			})
			return
		}
	}
	
	// Success message
	app.QueueUpdateDraw(func() {
		statusText.SetText(fmt.Sprintf("[green]Results exported to: %s", filePath))
	})
}

// GetDefaultCSVExportPath returns the default path for exporting CSV files
func GetDefaultCSVExportPath() string {
	// Try to use user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if we can't get home dir
		return fmt.Sprintf("snmp_connectivity_test_%s.csv", time.Now().Format("20060102_150405"))
	}
	
	// Create a timestamped filename in the user's home directory
	return fmt.Sprintf("%s/snmp_connectivity_test_%s.csv", homeDir, time.Now().Format("20060102_150405"))
}
