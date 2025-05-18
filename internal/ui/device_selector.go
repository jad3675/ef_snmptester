package ui

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"sync" // Added for parallel device testing
	"time"

	"github.com/jad3675/ef_snmptester/internal/model"
	"github.com/jad3675/ef_snmptester/internal/snmp"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// DeviceGroupResult represents a result of testing a device group
type DeviceGroupResult struct {
	DeviceName      string
	DeviceIP        string
	DeviceStatus    string
	GroupName       string
	Status          string
	OIDsRetrieved   string
	Duration        time.Duration
	SuccessPercent  float64
	TotalOIDs       int
	SuccessfulOIDs  int
	ErrorMessage    string
	WalkedOidData   map[string]map[string]interface{} // For detailed view
	SourceFile      string  // Moved to the end
}

// createDeviceSelector creates the device selector screen
func (a *App) createDeviceSelector() tview.Primitive {
	// Create a table for devices
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite))
	
	// Add headers
	table.SetCell(0, 0, tview.NewTableCell("Device Name").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetAlign(tview.AlignLeft))
	table.SetCell(0, 1, tview.NewTableCell("IP Address").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetAlign(tview.AlignLeft))
	table.SetCell(0, 2, tview.NewTableCell("Source File").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetAlign(tview.AlignLeft))
	
	// Create status text view
	statusText := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("")
	
	// Create search input field
	searchField := tview.NewInputField().
		SetLabel("Search: ").
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(tcell.ColorWhite).
		SetLabelColor(tcell.ColorYellow)
	
	// Content layout without search
	normalFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(statusText, 1, 1, false).
		AddItem(table, 0, 1, true)
	
	// Content layout with search
	searchFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(searchField, 1, 1, true).
		AddItem(statusText, 1, 1, false).
		AddItem(table, 0, 1, false)
		
	// Pages for switching between normal and search mode
	contentPages := tview.NewPages().
		AddPage("normal", normalFlex, true, true).
		AddPage("search", searchFlex, true, false)
	
	// Main flex container 
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetText("ElastiFlow SNMP Tester - Select a Device"), 1, 1, false).
		AddItem(contentPages, 0, 1, true).
		AddItem(tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignCenter).
			SetText("[yellow]ESC[white]: Back to menu    [yellow]Enter[white]: Select device    [yellow]a[white]: Test all devices    [yellow]s[white]: Test displayed devices    [yellow]/[white]: Search"), 1, 1, false)
	
	// Track current filter and filtered devices
	var currentFilter string
	var displayedDevices []*model.Device
	
	// Function to populate the table with all devices or filtered devices
	populateTable := func(filter string) {
		// Clear table except headers
		table.Clear()
		
		// Re-add headers
		table.SetCell(0, 0, tview.NewTableCell("Device Name").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
		table.SetCell(0, 1, tview.NewTableCell("IP Address").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
		table.SetCell(0, 2, tview.NewTableCell("Source File").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
		
		// Update the current filter
		currentFilter = filter
		
		// Get device names for iteration
		deviceNames := make([]string, 0, len(a.configManager.Devices))
		for name := range a.configManager.Devices {
			deviceNames = append(deviceNames, name)
		}
		
		// Reset the displayed devices list
		displayedDevices = nil
		
		// Add devices to the table
		row := 1
		matchCount := 0
		
		for _, name := range deviceNames {
			device := a.configManager.Devices[name]
			
			// Apply filter if provided
			if filter != "" {
				filterLower := strings.ToLower(filter)
				nameMatch := strings.Contains(strings.ToLower(name), filterLower)
				ipMatch := strings.Contains(strings.ToLower(device.IP), filterLower)
				sourceMatch := strings.Contains(strings.ToLower(device.SourceFile), filterLower)
				
				if !nameMatch && !ipMatch && !sourceMatch {
					continue // Skip this device as it doesn't match the filter
				}
			}
			
			// Add this device to the displayed list
			displayedDevices = append(displayedDevices, device)
			
			matchCount++
			
			table.SetCell(row, 0, tview.NewTableCell(name).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft))
			table.SetCell(row, 1, tview.NewTableCell(device.IP).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft))
			table.SetCell(row, 2, tview.NewTableCell(device.SourceFile).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft))
			row++
		}
		
		// Update status with filter info
		if filter != "" {
			statusText.SetText(fmt.Sprintf("Filtered by: '%s' - %d device(s) matching", filter, matchCount))
		} else {
			statusText.SetText("")
		}
		
		// Make sure the first row is selected if any devices exist
		if row > 1 {
			table.Select(1, 0)
		}
	}
	
	// Variable to track search mode
	var isSearchMode bool
	
	// Set up search field handler
	searchField.SetChangedFunc(func(text string) {
		// Apply filter on each keystroke
		populateTable(text)
	})
	
	searchField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			// Apply current filter and switch focus to table
			contentPages.SwitchToPage("normal")
			isSearchMode = false
			a.app.SetFocus(table)
		case tcell.KeyEscape:
			// Cancel search, clear filter, and switch focus to table
			searchField.SetText("")
			populateTable("")
			contentPages.SwitchToPage("normal")
			isSearchMode = false
			a.app.SetFocus(table)
		}
	})
	
	// Initially populate with all devices
	populateTable("")
	
	// Set up device selection handling
	table.Select(1, 0) // Select the first device row
	table.SetSelectedFunc(func(row int, column int) {
		if row > 0 { // Skip header row
			deviceName := table.GetCell(row, 0).Text
			device := a.configManager.Devices[deviceName]
			groupSelector := a.createGroupSelector(device)
			a.pages.AddPage("group_selector", groupSelector, true, true)
			a.pages.SwitchToPage("group_selector")
		}
	})
	
	// Common function to create test results page and run tests
	testDevicesFunc := func(devices []*model.Device, testTitle string) {
		if len(devices) == 0 {
			statusText.SetText("[red]No devices to test")
			return
		}
		
		statusText.SetText(fmt.Sprintf("[yellow]Preparing to test %d device(s)...", len(devices)))
		
		// Create results table
		resultsTable := tview.NewTable().
			SetBorders(false).
			SetSelectable(true, false) // Make rows selectable

		// Add headers
		resultsTable.SetCell(0, 0, tview.NewTableCell("Device").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
		resultsTable.SetCell(0, 1, tview.NewTableCell("Group").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
		resultsTable.SetCell(0, 2, tview.NewTableCell("Status").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
		resultsTable.SetCell(0, 3, tview.NewTableCell("OIDs Retrieved").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
		resultsTable.SetCell(0, 4, tview.NewTableCell("Time (ms)").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
		resultsTable.SetCell(0, 5, tview.NewTableCell("Source File").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
		
		// Create status text for results
		resultsStatusText := tview.NewTextView().
			SetDynamicColors(true).
			SetText("Starting tests...")
		
		// Create results flex
		resultsFlex := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(tview.NewTextView().
				SetTextAlign(tview.AlignCenter).
				SetText(fmt.Sprintf("ElastiFlow SNMP Tester - %s", testTitle)), 1, 1, false).
			AddItem(resultsStatusText, 1, 1, false).
			AddItem(resultsTable, 0, 1, true).
			AddItem(tview.NewTextView().
				SetDynamicColors(true).
				SetTextAlign(tview.AlignCenter).
				SetText("[yellow]ESC[white]: Back to device selection    [yellow]e[white]: Export results to CSV"), 1, 1, false)
		
		// Results storage for export
		var allResults []DeviceGroupResult // This is populated by the main test goroutine
		
		resultsTable.SetSelectedFunc(func(row, column int) {
			if row > 0 && row <= len(allResults) { // row is 1-based, skip header
				selectedResult := allResults[row-1] // allResults is 0-based
				
				detailView := a.createDetailedResultView(selectedResult)
				pageName := "detailed_result_view"
				a.pages.AddPage(pageName, detailView, true, true)
				// No need to SwitchToPage here, AddPage with last param true does it.
			}
		})

		// Set up key bindings for results (ESC and 'e' for export)
		resultsFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEscape {
				a.SwitchToPage("device_selector")
				return nil
			} else if event.Key() == tcell.KeyRune && event.Rune() == 'e' && len(allResults) > 0 {
				// Export results to CSV
				filePath := fmt.Sprintf("%s/snmp_all_tests_%s.csv", getHomeDir(), 
					time.Now().Format("20060102_150405"))
				
				// Run export in goroutine to avoid UI blocking
				go func() {
					err := exportAllResultsToCSV(allResults, filePath)
					if err != nil {
						a.app.QueueUpdateDraw(func() {
							resultsStatusText.SetText(fmt.Sprintf("[red]Export failed: %s", err))
						})
					} else {
						a.app.QueueUpdateDraw(func() {
							resultsStatusText.SetText(fmt.Sprintf("[green]Results exported to: %s", filePath))
						})
					}
				}()
				return nil
			}
			return event
		})
		
		// Create frame for results
		resultsFrame := tview.NewFrame(resultsFlex).
			SetBorders(0, 0, 0, 0, 0, 0)
		
		// Add results page
		a.pages.AddPage("all_devices_results", resultsFrame, true, true)
		a.pages.SwitchToPage("all_devices_results")
		
		// Run tests in goroutine
		go func() {
			const maxConcurrentDeviceTests = 5 // Max number of devices to test in parallel
			var wg sync.WaitGroup
			deviceTestSemaphore := make(chan struct{}, maxConcurrentDeviceTests)
			deviceGroupResultChan := make(chan DeviceGroupResult, len(devices)*5) // Buffer size guess
			
			// allResults is already declared in the outer scope and captured by this closure.
			// We will append directly to it. Ensure it's clear if tests are re-run.
			// However, allResults is re-assigned later, so let's use a local slice for collection first, then assign.
			// This matches the existing logic for allResults scope.
			collectedDeviceGroupResults := []DeviceGroupResult{}


			totalGroupsToTest := 0
			for _, d := range devices {
				totalGroupsToTest += len(d.DeviceGroups)
			}
			groupsProcessed := 0

			// Launch per-device testing goroutines
			for _, deviceToTest := range devices {
				wg.Add(1)
				deviceTestSemaphore <- struct{}{} // Acquire semaphore

				go func(d *model.Device) {
					defer wg.Done()
					defer func() { <-deviceTestSemaphore }() // Release semaphore

					for _, groupName := range d.DeviceGroups {
						// Update interim status
						a.app.QueueUpdateDraw(func() {
							groupsProcessed++
							resultsStatusText.SetText(fmt.Sprintf("Testing group %s for device %s... (%d/%d groups processed)",
								groupName, d.Name, groupsProcessed, totalGroupsToTest))
						})

						// Get OIDs for this group
						oids, err := a.configManager.GetDeviceOIDs(d.Name, groupName)
						if err != nil {
							deviceGroupResultChan <- DeviceGroupResult{
								DeviceName:    d.Name, DeviceIP: d.IP, DeviceStatus: "Failed", GroupName: groupName,
								Status: "Failed", OIDsRetrieved: "0/0", Duration: 0,
								ErrorMessage: fmt.Sprintf("Failed to get OIDs: %s", err), SourceFile: d.SourceFile,
								WalkedOidData: nil, // No walk data if OIDs couldn't be retrieved
							}
							continue
						}

						// Create SNMP client
						client, err := snmp.NewClient(d)
						if err != nil {
							deviceGroupResultChan <- DeviceGroupResult{
								DeviceName:    d.Name, DeviceIP: d.IP, DeviceStatus: "Failed", GroupName: groupName,
								Status: "Failed", OIDsRetrieved: "0/0", Duration: 0,
								ErrorMessage: fmt.Sprintf("Failed to create SNMP client: %s", err), SourceFile: d.SourceFile,
								WalkedOidData: nil, // No walk data if client creation failed
							}
							continue
						}

						// Test the group
						startTime := time.Now()
						testResult, _ := client.TestGroupOIDs(oids)
						duration := time.Since(startTime)

						// Calculate success percentage
						totalOIDs := 0
						successfulOIDs := 0
						if val, ok := testResult.Data["total_oids"]; ok { fmt.Sscanf(val, "%d", &totalOIDs) }
						if val, ok := testResult.Data["successful_oids"]; ok { fmt.Sscanf(val, "%d", &successfulOIDs) }
						
						var successPercent float64 = 0
						if totalOIDs > 0 { successPercent = float64(successfulOIDs) / float64(totalOIDs) * 100 }
						groupStatus := string(testResult.Status)

						deviceGroupResultChan <- DeviceGroupResult{
							DeviceName: d.Name, DeviceIP: d.IP, GroupName: groupName, Status: groupStatus,
							OIDsRetrieved: fmt.Sprintf("%s/%s", testResult.Data["successful_oids"], testResult.Data["total_oids"]),
							Duration: duration, SuccessPercent: successPercent, TotalOIDs: totalOIDs, SuccessfulOIDs: successfulOIDs,
							ErrorMessage: testResult.Message, SourceFile: d.SourceFile,
							WalkedOidData: testResult.WalkedOidData, // Copy the detailed walk data
						}
					}
				}(deviceToTest)
			}

			// Goroutine to close result channel when all device tests are done
			go func() {
				wg.Wait()
				close(deviceGroupResultChan)
			}()

			// Collect results and update table incrementally
			for res := range deviceGroupResultChan {
				collectedDeviceGroupResults = append(collectedDeviceGroupResults, res)

				// Incremental table update
				a.app.QueueUpdateDraw(func() {
					newRowIndex := resultsTable.GetRowCount()
					
					var statusColor tcell.Color
					if res.Status == "success" { statusColor = tcell.ColorGreen } else { statusColor = tcell.ColorRed }

					resultsTable.SetCell(newRowIndex, 0, tview.NewTableCell(res.DeviceName).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
					resultsTable.SetCell(newRowIndex, 1, tview.NewTableCell(res.GroupName).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
					resultsTable.SetCell(newRowIndex, 2, tview.NewTableCell(res.Status).SetTextColor(statusColor).SetAlign(tview.AlignLeft)) // Group status
					resultsTable.SetCell(newRowIndex, 3, tview.NewTableCell(res.OIDsRetrieved).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
					resultsTable.SetCell(newRowIndex, 4, tview.NewTableCell(fmt.Sprintf("%d", res.Duration.Milliseconds())).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
					resultsTable.SetCell(newRowIndex, 5, tview.NewTableCell(res.SourceFile).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
					resultsTable.ScrollToEnd()
				})
			}
			
			// All tests complete, now process collected results for final summary
			allResults = collectedDeviceGroupResults // For export

			// Calculate device status and counts
			deviceSuccessStatus := make(map[string]struct{
				hasSuccessfulGroups bool
				processedAllGroups  bool // True if at least one group result exists for the device
			})
			deviceGroupCounts := make(map[string]int) // Track how many groups processed per device

			for _, res := range collectedDeviceGroupResults {
				status, exists := deviceSuccessStatus[res.DeviceName]
				if !exists {
					status = struct{hasSuccessfulGroups bool; processedAllGroups  bool}{false, false}
				}
				status.processedAllGroups = true // Mark as processed
				if res.Status == "success" {
					status.hasSuccessfulGroups = true
				}
				deviceSuccessStatus[res.DeviceName] = status
				deviceGroupCounts[res.DeviceName]++
			}
			
			deviceSuccessCount := 0
			totalDevicesProcessed := 0 // Count devices for which we have at least one group result

			for _, status := range deviceSuccessStatus {
				// A device is processed if we have at least one group result for it.
				// This logic assumes all defined groups for a device are attempted.
				if status.processedAllGroups {
					totalDevicesProcessed++
					if status.hasSuccessfulGroups {
						deviceSuccessCount++
					}
				}
			}
			
			// Update DeviceStatus in collectedDeviceGroupResults
			for i := range collectedDeviceGroupResults {
				deviceName := collectedDeviceGroupResults[i].DeviceName
				if status, exists := deviceSuccessStatus[deviceName]; exists && status.processedAllGroups {
					if status.hasSuccessfulGroups {
						collectedDeviceGroupResults[i].DeviceStatus = "Success"
					} else {
						collectedDeviceGroupResults[i].DeviceStatus = "Failed"
					}
				} else {
					collectedDeviceGroupResults[i].DeviceStatus = "Failed" // Default if no groups processed or device not found
				}
			}

			successfulGroups := 0
			totalDuration := time.Duration(0)
			for _, res := range collectedDeviceGroupResults {
				if res.Status == "success" {
					successfulGroups++
				}
				totalDuration += res.Duration
			}

			// Update Final Status Text (Table is already populated)
			a.app.QueueUpdateDraw(func() {
				// Update the status message
				// Use len(devices) for total devices expected, as totalDevicesProcessed might be lower if a device has no groups.
				resultsStatusText.SetText(fmt.Sprintf(
					"Testing complete. %d of %d devices successful. %d of %d groups successful. Total time: %d ms. Press 'e' to export results.",
					deviceSuccessCount, len(devices), successfulGroups, len(collectedDeviceGroupResults), totalDuration.Milliseconds()))
			})
		}()
	}
	
	// Function to test all devices and their groups
	testAllDevicesAndGroups := func() {
		// Get all devices as a slice
		allDevices := make([]*model.Device, 0, len(a.configManager.Devices))
		for _, device := range a.configManager.Devices {
			allDevices = append(allDevices, device)
		}
		
		// Run test with all devices
		testDevicesFunc(allDevices, "All Devices/Groups Results")
	}
	
	// Function to test only displayed devices
	testDisplayedDevices := func() {
		if len(displayedDevices) == 0 {
			statusText.SetText("[red]No devices to test")
			return
		}
		
		// Create a descriptive title based on the filter
		var title string
		if currentFilter != "" {
			title = fmt.Sprintf("Filtered Devices Results (filter: '%s')", currentFilter)
		} else {
			title = "All Devices/Groups Results"
		}
		
		// Run test with displayed devices only
		testDevicesFunc(displayedDevices, title)
	}
	
	// Set up key bindings for the flex container
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Check if search is active
		if isSearchMode {
			// Let search field handle the event
			return event
		}
		
		// Otherwise handle main key events
		if event.Key() == tcell.KeyEscape {
			a.SwitchToPage("main")
			return nil
		} else if event.Key() == tcell.KeyEnter {
			row, _ := table.GetSelection()
			if row > 0 { // Skip header row
				deviceName := table.GetCell(row, 0).Text
				device := a.configManager.Devices[deviceName]
				groupSelector := a.createGroupSelector(device)
				a.pages.AddPage("group_selector", groupSelector, true, true)
				a.pages.SwitchToPage("group_selector")
			}
			return nil
		} else if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'a':
				testAllDevicesAndGroups()
				return nil
			case 's':
				testDisplayedDevices()
				return nil
			case '/': // Activate search
				searchField.SetText("")
				contentPages.SwitchToPage("search")
				isSearchMode = true
				a.app.SetFocus(searchField)
				return nil
			}
		}
		return event
	})
	
	return flex
}

// createGroupSelector creates the group selector screen for a device
func (a *App) createGroupSelector(device *model.Device) tview.Primitive {
	// Create the list
	list := tview.NewList().
		SetHighlightFullLine(true).
		SetWrapAround(false).
		SetSelectedBackgroundColor(tcell.ColorBlue)
	
	// Add device groups to the list
	for _, groupName := range device.DeviceGroups {
		list.AddItem(groupName, "", 0, nil)
	}
	
	// Create a frame for the list
	frame := tview.NewFrame(list).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText(fmt.Sprintf("ElastiFlow SNMP Tester - Device: %s - Group Selection - Source: %s", 
			device.Name, device.SourceFile), true, tview.AlignCenter, tcell.ColorBlue).
		AddText("ESC: Back to device selection    Enter: Select group", false, tview.AlignCenter, tcell.ColorWhite)
	
	// Set up group selection handling
	list.SetSelectedFunc(func(index int, groupName string, description string, shortcut rune) {
		groupTest := a.createGroupTest(device, groupName)
		a.pages.AddPage("group_test", groupTest, true, true)
		a.pages.SwitchToPage("group_test")
	})
	
	// Set up key bindings
	frame.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.SwitchToPage("device_selector")
			return nil
		}
		return event
	})
	
	return frame
}

// exportAllResultsToCSV exports all device and group test results to a CSV file
func exportAllResultsToCSV(results []DeviceGroupResult, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Write header
	if err := writer.Write([]string{
		"Device Name",
		"Device IP",
		"Device Status",
		"Group Name",
		"Group Status",
		"OIDs Retrieved",
		"Total OIDs",
		"Successful OIDs",
		"Success %",
		"Duration (ms)",
		"Error Message",
		"Source File", // Moved to the end
	}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	
	// Write data rows
	for _, result := range results {
		if err := writer.Write([]string{
			result.DeviceName,
			result.DeviceIP,
			result.DeviceStatus,
			result.GroupName,
			result.Status,
			result.OIDsRetrieved,
			fmt.Sprintf("%d", result.TotalOIDs),
			fmt.Sprintf("%d", result.SuccessfulOIDs),
			fmt.Sprintf("%.2f", result.SuccessPercent),
			fmt.Sprintf("%d", result.Duration.Milliseconds()),
			result.ErrorMessage,
			result.SourceFile, // Moved to the end
		}); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}
	
	return nil
}

// getHomeDir returns the user's home directory or current directory if not found
func getHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return homeDir
}

// createDetailedResultView creates a new page to display detailed results for a single DeviceGroupResult
func (a *App) createDetailedResultView(result DeviceGroupResult) tview.Primitive {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	var content strings.Builder
	content.WriteString(fmt.Sprintf("[yellow]Device:[white] %s (%s)\n", result.DeviceName, result.DeviceIP))
	content.WriteString(fmt.Sprintf("[yellow]Group:[white]  %s\n", result.GroupName))
	content.WriteString(fmt.Sprintf("[yellow]Status:[white] %s\n", result.Status))
	content.WriteString(fmt.Sprintf("[yellow]OIDs: [white]  %s\n", result.OIDsRetrieved))
	content.WriteString(fmt.Sprintf("[yellow]Time:[white]   %s\n", result.Duration.String()))
	if result.ErrorMessage != "" {
		content.WriteString(fmt.Sprintf("[yellow]Error:[white]  [red]%s[white]\n", result.ErrorMessage))
	}
	content.WriteString("\n[yellow]Walked OID Data:\n")

	if len(result.WalkedOidData) == 0 {
		content.WriteString("  [grey]No OID data walked or available.\n")
	} else {
		for oidName, walkMap := range result.WalkedOidData {
			content.WriteString(fmt.Sprintf("  [cyan]OID Name: %s[white]\n", oidName))
			if errVal, isErr := walkMap["error"]; isErr {
				content.WriteString(fmt.Sprintf("    [red]Error: %v[white]\n", errVal))
			} else if len(walkMap) == 0 {
				content.WriteString("    [grey]No data retrieved for this OID.\n")
			} else {
				for instance, value := range walkMap {
					content.WriteString(fmt.Sprintf("    %s = %v\n", instance, value))
				}
			}
		}
	}

	textView.SetText(content.String())

	frame := tview.NewFrame(textView).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText(fmt.Sprintf("Detailed Result - %s / %s", result.DeviceName, result.GroupName), true, tview.AlignCenter, tcell.ColorBlue).
		AddText("ESC: Back to Results", false, tview.AlignCenter, tcell.ColorWhite)

	frame.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.pages.SwitchToPage("all_devices_results") // Assuming this is the name of the results page
			a.pages.RemovePage("detailed_result_view")  // Clean up the page
			return nil
		}
		return event
	})

	return frame
}
