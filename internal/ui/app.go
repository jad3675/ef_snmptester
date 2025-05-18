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
	
	// Walk OIDs
	walkResults, err := client.WalkOIDs(oids)
	if err != nil {
		a.app.QueueUpdateDraw(func() {
			statusText.SetText(fmt.Sprintf("[red]Error walking OIDs: %s", err))
			*testing = false
		})
		return
	}
	
	// Store results
	*results = walkResults
	
	// Count successful OIDs
	successful := 0
	for _, resultMap := range walkResults {
		if _, ok := resultMap["error"]; !ok && len(resultMap) > 0 {
			successful++
		}
	}
	
	// Update the table with results
	a.app.QueueUpdateDraw(func() {
		updateGroupTestResults(walkResults, oids, table)
		
		// Update status text
		statusText.SetText(fmt.Sprintf("Testing complete for %s in group %s. %d of %d OIDs successfully retrieved.",
			device.Name, groupName, successful, len(oids)))
		
		*testing = false
	})
}

// updateGroupTestResults updates the group test results in the table
func updateGroupTestResults(
	results map[string]map[string]interface{},
	oids map[string]string,
	table *tview.Table,
) {
	for oidName := range oids {
		result, ok := results[oidName]
		
		// Find the row for this OID
		var oidRow int
		for r := 1; r < table.GetRowCount(); r++ {
			nameCell := table.GetCell(r, 0)
			if nameCell != nil && nameCell.Text == oidName {
				oidRow = r
				break
			}
		}
		
		if oidRow == 0 {
			continue // OID not found in table
		}
		
		// Update status
		if ok {
			// Check if there was an error
			if _, hasError := result["error"]; hasError {
				table.SetCell(oidRow, 2, tview.NewTableCell("Failed").
					SetTextColor(tcell.ColorRed).
					SetAlign(tview.AlignLeft))
			} else if len(result) > 0 {
				table.SetCell(oidRow, 2, tview.NewTableCell(fmt.Sprintf("Success (%d values)", len(result))).
					SetTextColor(tcell.ColorGreen).
					SetAlign(tview.AlignLeft))
			} else {
				table.SetCell(oidRow, 2, tview.NewTableCell("No Data").
					SetTextColor(tcell.ColorYellow).
					SetAlign(tview.AlignLeft))
			}
		} else {
			table.SetCell(oidRow, 2, tview.NewTableCell("Not Found").
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignLeft))
		}
	}
}
