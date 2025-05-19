// Package ui provides a terminal user interface
package ui

import (
	"fmt"
	
	"github.com/jad3675/ef_snmptester/internal/config"
	"github.com/jad3675/ef_snmptester/internal/model"
	"github.com/jad3675/ef_snmptester/internal/snmp"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// App is the main UI application
type App struct {
	app           *tview.Application
	pages         *tview.Pages
	configManager *config.Manager
	configPath    string
}

// NewApp creates a new UI application
func NewApp(configPath string) *App {
	app := &App{
		app:           tview.NewApplication(),
		pages:         tview.NewPages(),
		configManager: config.NewManager(configPath),
		configPath:    configPath,
	}
	
	return app
}

// Start initializes and starts the UI application
func (a *App) Start() error {
	// Load configuration
	if err := a.configManager.LoadAll(); err != nil {
		return err
	}
	
	// Initialize main menu
	mainMenu := a.createMainMenu()
	a.pages.AddPage("main", mainMenu, true, true)
	
	// Set up the application
	a.app.SetRoot(a.pages, true)
	
	// Run the application
	return a.app.Run()
}

// Stop stops the application
func (a *App) Stop() {
	a.app.Stop()
}

// SwitchToPage switches to a specific page
func (a *App) SwitchToPage(name string) {
	a.pages.SwitchToPage(name)
}

// createMainMenu creates the main menu
func (a *App) createMainMenu() tview.Primitive {
	// Create the menu list
	list := tview.NewList().
		AddItem("Test SNMP Connectivity", "Test basic SNMP connectivity to all configured devices", 'c', func() {
			connectivityTest := a.createConnectivityTest()
			a.pages.AddPage("connectivity", connectivityTest, true, true)
			a.pages.SwitchToPage("connectivity")
		}).
		AddItem("Test Device Group", "Test SNMP data collection for specific device groups", 'g', func() {
			deviceSelector := a.createDeviceSelector()
			a.pages.AddPage("device_selector", deviceSelector, true, true)
			a.pages.SwitchToPage("device_selector")
		}).
		AddItem("Quit", "Exit the application", 'q', func() {
			a.Stop()
		})
	
	// Create a frame for the menu
	frame := tview.NewFrame(list).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText("ElastiFlow SNMP Tester", true, tview.AlignCenter, tcell.ColorBlue).
		AddText("Use arrow keys to navigate, Enter to select", false, tview.AlignCenter, tcell.ColorWhite)
	
	// Set up key bindings
	frame.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyCtrlC {
			a.Stop()
			return nil
		}
		return event
	})
	
	return frame
}

// createConnectivityTest creates the connectivity test screen
// Implemented in connectivity.go

// createDeviceSelector creates the device selector screen
// Implemented in device_selector.go

// createGroupSelector creates the group selector screen for a device
// Implemented in device_selector.go

// createGroupTest creates the group test screen for a device and group
func (a *App) createGroupTest(device *model.Device, groupName string) tview.Primitive {
	// Create the table
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	
	// Add headers
	table.SetCell(0, 0, tview.NewTableCell("OID Name").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetAlign(tview.AlignLeft))
	table.SetCell(0, 1, tview.NewTableCell("OID").
		SetTextColor(tcell.ColorYellow).
		SetSelectable(false).
		SetAlign(tview.AlignLeft))
	table.SetCell(0, 2, tview.NewTableCell("Status").
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
			SetText(fmt.Sprintf("ElastiFlow SNMP Test - Device: %s - Group: %s - Source: %s", 
				device.Name, groupName, device.SourceFile)), 1, 1, false).
		AddItem(statusText, 1, 1, false).
		AddItem(table, 0, 1, true).
		AddItem(tview.NewTextView().
			SetDynamicColors(true).
			SetText("[yellow]ESC[white]: Back to group selection    [yellow]r[white]: Restart test"), 1, 1, false)
	
	// Create a frame for the flex layout
	frame := tview.NewFrame(flex).
		SetBorders(0, 0, 0, 0, 0, 0)
	
	// Testing state
	var (
		testing bool
		results map[string]map[string]interface{}
	)
	
	// Function to start the test
	startTest := func() {
		if testing {
			return
		}
		
		testing = true
		results = make(map[string]map[string]interface{})
		
		// Clear existing table rows (except headers)
		table.Clear()
		// Re-add headers
		table.SetCell(0, 0, tview.NewTableCell("OID Name").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
		table.SetCell(0, 1, tview.NewTableCell("OID").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
		table.SetCell(0, 2, tview.NewTableCell("Status").
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft))
		
		// Get OIDs for the device group
		oids, err := a.configManager.GetDeviceOIDs(device.Name, groupName)
		if err != nil {
			statusText.SetText(fmt.Sprintf("[red]Error getting OIDs: %s", err))
			testing = false
			return
		}
		
		// Add rows for each OID with "Pending" status
		row := 1
		for oidName, oid := range oids {
			// Add OID row
			table.SetCell(row, 0, tview.NewTableCell(oidName).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft))
			table.SetCell(row, 1, tview.NewTableCell(oid).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignLeft))
			table.SetCell(row, 2, tview.NewTableCell("Pending").
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft))
			
			row++
		}
		
		statusText.SetText(fmt.Sprintf("Testing device %s with group %s...", device.Name, groupName))
		
		// Run test in background
		go a.runGroupTest(device, groupName, oids, table, statusText, &testing, &results)
	}
	
	// Set up key bindings
	frame.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.SwitchToPage("group_selector")
			return nil
		} else if event.Key() == tcell.KeyRune && event.Rune() == 'r' && !testing {
			startTest()
			return nil
		}
		return event
	})
	
	// Start the test automatically
	startTest()
	
	return frame
}

// runGroupTest runs an SNMP test for a device group
func (a *App) runGroupTest(
	device *model.Device,
	groupName string,
	oids map[string]string,
	table *tview.Table,
	statusText *tview.TextView,
	testing *bool,
	results *map[string]map[string]interface{},
) {
	// Create SNMP client
	client, err := snmp.NewClient(device)
	if err != nil {
		a.app.QueueUpdateDraw(func() {
			statusText.SetText(fmt.Sprintf("[red]Error creating SNMP client: %s", err))
			*testing = false
		})
		return
	}
	
	// Test OID Names using GetNext
	testResult, err := client.TestGroupOIDs(oids) // This now uses GetNext internally
	if err != nil {
		a.app.QueueUpdateDraw(func() {
			statusText.SetText(fmt.Sprintf("[red]Error testing OID group: %s", err))
			*testing = false
		})
		return
	}

	// Store results (WalkedOidData from testResult contains the GetNext results)
	*results = testResult.WalkedOidData
	
	// Count successful OID Name retrievals
	successfulOidNameCount := 0
	if valStr, ok := testResult.Data["successful_oids"]; ok {
		fmt.Sscanf(valStr, "%d", &successfulOidNameCount)
	}
	
	// Update the table with results
	a.app.QueueUpdateDraw(func() {
		// Pass testResult.WalkedOidData to updateGroupTestResults
		updateGroupTestResults(testResult.WalkedOidData, oids, table)
		
		// Update status text (using the message from testResult which is already formatted)
		statusText.SetText(fmt.Sprintf("Testing complete for %s in group %s. %s.",
			device.Name, groupName, testResult.Message))
		
		*testing = false
	})
}

// updateGroupTestResults updates the group test results in the table for GetNext results
func updateGroupTestResults(
	getNextResults map[string]map[string]interface{}, // Renamed to reflect new data
	oidNameMap map[string]string, // Map of OID Name -> OID String
	table *tview.Table,
) {
	for oidName := range oidNameMap {
		dataMap, ok := getNextResults[oidName]
		
		// Find the row for this OID Name
		var oidRow int
		for r := 1; r < table.GetRowCount(); r++ {
			nameCell := table.GetCell(r, 0)
			if nameCell != nil && nameCell.Text == oidName {
				oidRow = r
				break
			}
		}
		
		if oidRow == 0 {
			continue // OID Name not found in table (should not happen if table populated correctly)
		}
		
		// Update status cell (column 2)
		if ok {
			if errVal, hasError := dataMap["error"]; hasError {
				// Display the specific error for this OID Name
				table.SetCell(oidRow, 2, tview.NewTableCell(fmt.Sprintf("Failed: %v", errVal)).
					SetTextColor(tcell.ColorRed).
					SetAlign(tview.AlignLeft))
			} else if _, hasValue := dataMap["value_raw"]; hasValue {
				// Successfully got a value for this OID Name
				table.SetCell(oidRow, 2, tview.NewTableCell("Success (1 value)"). // Always 1 value for GetNext
					SetTextColor(tcell.ColorGreen).
					SetAlign(tview.AlignLeft))
			} else {
				// No error, but no value_raw key (unexpected state)
				table.SetCell(oidRow, 2, tview.NewTableCell("No Data").
					SetTextColor(tcell.ColorYellow).
					SetAlign(tview.AlignLeft))
			}
		} else {
			// OID Name was not in getNextResults map (should not happen if GetFirstEntryForOidNames processes all)
			table.SetCell(oidRow, 2, tview.NewTableCell("Not Processed").
				SetTextColor(tcell.ColorDarkGrey).
				SetAlign(tview.AlignLeft))
		}
	}
}
