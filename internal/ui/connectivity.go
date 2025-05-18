package ui

import (
	"fmt"
	"sync"
	
	"github.com/jad3675/ef_snmptester/internal/model"
	"github.com/jad3675/ef_snmptester/internal/snmp"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// createConnectivityTest creates the connectivity test screen
func (a *App) createConnectivityTest() tview.Primitive {
	// Create the table
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	
	// Add headers
	table.SetCell(0, 0, tview.NewTableCell("Device Name").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetAlign(tview.AlignLeft))
	table.SetCell(0, 1, tview.NewTableCell("IP Address").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetAlign(tview.AlignLeft))
	table.SetCell(0, 2, tview.NewTableCell("Status").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetAlign(tview.AlignLeft))
	table.SetCell(0, 3, tview.NewTableCell("Message").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetAlign(tview.AlignLeft))
	table.SetCell(0, 4, tview.NewTableCell("Source File").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetAlign(tview.AlignLeft))
	
	// Create status text view
	statusText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("Press 'r' to start test")
	
	// Create flex layout
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetText("ElastiFlow SNMP Connectivity Test"), 1, 1, false).
		AddItem(statusText, 1, 1, false).
		AddItem(table, 0, 1, true).
		AddItem(tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignCenter).
			SetText("[yellow]ESC[white]: Back to menu    [yellow]r[white]: Restart test    [yellow]e[white]: Export to CSV"), 1, 1, false)
	
	// Testing state
	var (
		isTestRunning bool  // Renamed from testing to avoid conflict
		testDone      chan bool
		results       []*model.TestResult
	)
	
	// Function to start the test
	startTest := func() {
		if isTestRunning {
			return
		}
		
		isTestRunning = true
		results = []*model.TestResult{}
		testDone = make(chan bool)
		
		// Clear existing table rows (except headers)
		// The table is re-populated from scratch, so we'll set the row count to 1 (keeping just the header)
		table.Clear().SetCell(0, 0, tview.NewTableCell("Device Name").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft)).
			SetCell(0, 1, tview.NewTableCell("IP Address").
				SetTextColor(tcell.ColorYellow).
				SetSelectable(false).
				SetAlign(tview.AlignLeft)).
			SetCell(0, 2, tview.NewTableCell("Status").
				SetTextColor(tcell.ColorYellow).
				SetSelectable(false).
				SetAlign(tview.AlignLeft)).
			SetCell(0, 3, tview.NewTableCell("Message").
				SetTextColor(tcell.ColorYellow).
				SetSelectable(false).
				SetAlign(tview.AlignLeft)).
			SetCell(0, 4, tview.NewTableCell("Source File").
				SetTextColor(tcell.ColorYellow).
				SetSelectable(false).
				SetAlign(tview.AlignLeft))
		
		// Add rows for each device with "Pending" status
		row := 1
		for name, device := range a.configManager.Devices {
			// Create result with pending status
			result := model.NewTestResult(name, device.IP, device.SourceFile, model.ConnectivityTest)
			result.Status = model.Pending
			results = append(results, result)
			
			// Add device row
			table.SetCell(row, 0, tview.NewTableCell(name).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft))
			table.SetCell(row, 1, tview.NewTableCell(device.IP).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft))
			table.SetCell(row, 2, tview.NewTableCell("Pending").
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft))
			table.SetCell(row, 3, tview.NewTableCell("").
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft))
			table.SetCell(row, 4, tview.NewTableCell(device.SourceFile).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft))
			
			row++
		}
		
		statusText.SetText("Testing SNMP connectivity to all devices...")
		
		// Run tests in background
		go runConnectivityTests(a, results, table, statusText, testDone, &isTestRunning)
	}
	
	// Main container
	pages := tview.NewPages()
	pages.AddPage("main", flex, true, true)
	
	// Set up key bindings using the App's global input capture
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.SwitchToPage("main")
			a.app.SetInputCapture(nil) // Clear this input capture when leaving
			return nil
		} else if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'r':
				if !isTestRunning {
					startTest()
				}
				return nil
			case 'e':
				if !isTestRunning && len(results) > 0 {
					// Export results
					filePath := GetDefaultCSVExportPath()
					statusText.SetText("Exporting results...")
					
					// Run in a separate goroutine to prevent blocking the UI
					go ExportConnectivityResultsToCSV(results, filePath, a.app, statusText)
				}
				return nil
			}
		}
		return event
	})
	
	// Start the test automatically
	startTest()
	
	return pages
}

// runConnectivityTests runs SNMP connectivity tests for all devices
func runConnectivityTests(a *App, results []*model.TestResult, table *tview.Table, statusText *tview.TextView, done chan bool, isTestRunning *bool) {
	var wg sync.WaitGroup
	resultChan := make(chan *model.TestResult, len(results))
	
	// Create a map to quickly find the original result by device name
	resultMap := make(map[string]*model.TestResult)
	for _, result := range results {
		resultMap[result.DeviceName] = result
	}
	
	// Test each device
	for _, result := range results {
		wg.Add(1)
		go func(result *model.TestResult) {
			defer wg.Done()
			
			device := a.configManager.Devices[result.DeviceName]
			
			// Create SNMP client
			client, err := snmp.NewClient(device)
			if err != nil {
				result.Failure(fmt.Sprintf("Failed to create SNMP client: %s", err), err)
				resultChan <- result
				return
			}
			
			// Test connectivity
			testResult, _ := client.TestConnectivity()
			resultChan <- testResult
		}(result)
	}
	
	// Wait for all tests to complete
	go func() {
		wg.Wait()
		close(resultChan)
		done <- true
	}()
	
	// Process results as they come in
	for result := range resultChan {
		// Update the original result in the results slice
		if origResult, ok := resultMap[result.DeviceName]; ok {
			origResult.Status = result.Status
			origResult.Message = result.Message
			origResult.Error = result.Error
			origResult.Data = result.Data
			origResult.EndTime = result.EndTime
			origResult.Duration = result.Duration
		}
		
		// Update the table
		updateConnectivityResult(result, table)
		
		// Force a redraw
		a.app.QueueUpdateDraw(func() {})
	}
	
	// Count successes and failures from the updated results
	successes := 0
	failures := 0
	for _, r := range results {
		if r.Status == model.Success {
			successes++
		} else if r.Status == model.Failure {
			failures++
		}
	}
	
	// Update status text
	a.app.QueueUpdateDraw(func() {
		statusText.SetText(fmt.Sprintf("Testing complete. %d device(s) tested: %d successful, %d failed. Press 'e' to export results.", 
			len(results), successes, failures))
	})
	
	// Set testing to false
	a.app.QueueUpdateDraw(func() {
		*isTestRunning = false
	})
}

// updateConnectivityResult updates a connectivity test result in the table
func updateConnectivityResult(result *model.TestResult, table *tview.Table) {
	// Find the row for this device
	for row := 1; row < table.GetRowCount(); row++ {
		nameCell := table.GetCell(row, 0)
		if nameCell != nil && nameCell.Text == result.DeviceName {
			// Update status
			statusText := string(result.Status)
			var statusColor tcell.Color
			
			switch result.Status {
			case model.Success:
				statusText = "Success"
				statusColor = tcell.ColorGreen
			case model.Failure:
				statusText = "Failed"
				statusColor = tcell.ColorRed
			case model.Pending:
				statusText = "Pending"
				statusColor = tcell.ColorYellow
			}
			
			table.SetCell(row, 2, tview.NewTableCell(statusText).
				SetTextColor(statusColor).
				SetAlign(tview.AlignLeft))
			
			// Update message
			message := result.Message
			if result.Error != nil {
				message = result.Error.Error()
			} else if result.Status == model.Success {
				if sysObjectID, ok := result.Data["sysObjectID"]; ok {
					message = fmt.Sprintf("sysObjectID: %s", sysObjectID)
				}
			}
			
			table.SetCell(row, 3, tview.NewTableCell(message).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft))
			
			return
		}
	}
}
