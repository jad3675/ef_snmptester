# ElastiFlow SNMP Tester

A Terminal User Interface (TUI) application for testing ElastiFlow SNMP configurations.

## Overview

ElastiFlow SNMP Tester is a Go-based TUI application that helps you test and troubleshoot your ElastiFlow SNMP collector configurations. Unlike the standard ElastiFlow SNMP collector, which requires you to start the full collector and sift through log files, this tool provides immediate feedback on your SNMP connectivity and MIB/OID responsiveness.

It supports testing basic SNMP connectivity to all configured devices and performing targeted SNMP GET-NEXT operations for OID Names within specific device groups to verify MIB reachability.

## Features

- **Connectivity Testing:** Test basic SNMP connectivity (sysObjectID GET) to all configured devices in parallel.
- **Device Group Testing (MIB Responsiveness Check):**
    - For each OID Name defined in a group, performs an SNMP GET-NEXT operation to retrieve the first lexicographical entry. This quickly verifies if the MIB/OID prefix is responsive.
    - Test specific device groups against individual devices.
    - Test all groups for all configured devices, or only for currently displayed/filtered devices.
    - **Parallelism:**
        - Device tests (when testing all/multiple devices) run in parallel (default up to 5 devices concurrently).
        - OID Name GET-NEXT operations *within* a single group test for a device run in parallel, respecting the device's `MaxConcurrentPolls` setting (defaults to 4 if not specified or invalid in YAML).
    - **Early Abort:** For unresponsive devices during a group test, after 5 consecutive OID Name query failures, remaining OID Names for that group test are skipped to save time.
- **Detailed Result View:** Inspect individual group test results, showing the outcome of the GET-NEXT operation for each OID Name (retrieved OID, type, value, or error message).
- **SNMP Support:** SNMPv2c and SNMPv3.
- **User Interface:** Easy-to-use terminal interface with real-time feedback and incremental table updates.
- **Device Search:** Filter the device list by name, IP, or source file.
- **Export Options:**
    - Export connectivity test results to CSV.
    - Export "All Devices/Groups" test results to CSV.
    - Export the "Detailed Result View" for a specific device-group test to JSON.

## Prerequisites

- Go 1.19 or higher (for building)
- ElastiFlow SNMP collector configuration files (YAML format)

## Quick Start

### Using the build script

The easiest way to build the application is by using the included build script:

```bash
chmod +x build.sh
./build.sh
```

This will build the application and provide instructions on how to run it.

### Manual build

If you prefer to build manually:

```bash
# Navigate to the project directory if you are not already there
# cd snmptester 
go mod tidy
go build -o snmptester ./cmd/snmptester
```

### Running the application

After building, you can run the application:

```bash
# Using the default configuration path (/etc/elastiflow/snmp)
./snmptester

# Using a custom configuration path
./snmptester -c /path/to/elastiflow/snmp
```

## Usage

```bash
snmptester [options]
```

### Options

- `-c, --config string`: Path to ElastiFlow SNMP configuration directory (default: "/etc/elastiflow/snmp")
- `-h, --help`: Show help
- `-v, --version`: Show version

## UI Navigation

### Main Menu

- Use arrow keys (↑/↓) to navigate the menu.
- Press `Enter` to select a menu item.
- Press `q` or `ESC` to quit the application from the main menu.

### Connectivity Test Screen

- Displays SNMP connectivity test results for all configured devices.
- Tests run in parallel for all devices.
- **Key Bindings:**
    - `ESC`: Return to the main menu.
    - `r`: Restart the connectivity test for all devices.
    - `e`: Export the current results to a CSV file.

### Device Group Testing Flow

1.  **Device Selection Screen (`Test Device Group` from Main Menu):**
    - Lists all configured devices.
    - **Key Bindings:**
        - `↑`/`↓`: Navigate the device list.
        - `Enter`: Select the highlighted device to proceed to its group selection.
        - `a`: Test all groups for **all** listed devices. This opens the "All Devices/Groups Results" screen.
        - `s`: Test all groups for **currently displayed/filtered** devices. This also opens the "All Devices/Groups Results" screen.
        - `/`: Enter search mode to filter the device list.
            - Type to filter by device name, IP, or source file.
            - `Enter` (in search mode): Apply filter and return to device list navigation.
            - `ESC` (in search mode): Clear filter and return to device list navigation.
        - `ESC`: Return to the main menu.

2.  **Group Selection Screen (after selecting a single device):**
    - Lists all device groups configured for the selected device.
    - **Key Bindings:**
        - `↑`/`↓`: Navigate the group list.
        - `Enter`: Select the highlighted group to test it against the device. This opens the "Single Group Test Results" screen.
        - `ESC`: Return to the device selection screen.

3.  **Single Group Test Results Screen:**
    - Displays SNMP GET-NEXT test results for each OID Name in the selected device and group. OID Names are queried in parallel, respecting the device's `MaxConcurrentPolls` setting (defaults to 4).
    - **Key Bindings:**
        - `ESC`: Return to the group selection screen for the current device.
        - `r`: Restart the test for this specific device and group.

4.  **All Devices/Groups Results Screen (after pressing 'a' or 's' on Device Selection):**
    - Displays results for multiple device-group tests (using GET-NEXT per OID Name).
    - Devices are tested in parallel (default up to 5 concurrently). OID Names within each group are also queried in parallel.
    - The table updates incrementally as results come in.
    - **Key Bindings:**
        - `↑`/`↓`: Navigate the results table.
        - `Enter`: On a selected result row, open the **Detailed Result View** for that specific device-group test.
        - `ESC`: Return to the device selection screen.
        - `e`: Export all displayed results to a CSV file.

5.  **Detailed Result View (after pressing Enter on a result in "All Devices/Groups Results" or from a Single Group Test if implemented):**
    - Shows detailed information for a single device-group test, including:
        - Device Name, IP, Group Name, Overall Group Status, OIDs Responded / Total OID Names, Duration, Error Message for the group.
        - **OID Name Results:** For each OID Name in the group:
            - The OID Name itself.
            - If an error occurred for this OID Name: The specific error message.
            - If successful: The exact "Retrieved OID", its "Type", and its "Value".
            - If no data was retrieved despite no error: A "No Data" or similar message.
    - **Key Bindings:**
        - `ESC`: Return to the previous results screen ("All Devices/Groups Results" or "Single Group Test Results").
        - `E`: Export the currently displayed detailed result to a JSON file.

## Troubleshooting

If you encounter any issues:

1.  Ensure the ElastiFlow SNMP configuration directory (e.g., `/etc/elastiflow/snmp` or your custom path) exists and contains valid `device.collection-*.yml` and `oid.collection-*.yml` files.
2.  Verify your SNMP device configurations within the YAML files (IP addresses, communities, SNMPv3 credentials, port numbers, `MaxConcurrentPolls`). The `MaxConcurrentPolls` setting (defaulting to 4 if not specified or invalid) controls parallelism for OID Name queries within a group test for a device.
3.  Make sure your network allows SNMP traffic (typically UDP port 161) to the target devices from the machine running `snmptester`.
4.  Check for typos in device names or group names referenced in your configurations.
5.  If a device is unresponsive, the group test for that device should skip remaining OID Names after 5 consecutive failures for that specific group test.

## License

MIT License

## Acknowledgements

- [ElastiFlow](https://github.com/robcowart/elastiflow) - Network flow monitoring and analytics solution
- [tcell](https://github.com/gdamore/tcell) - Terminal handling library
- [tview](https://github.com/rivo/tview) - Rich interactive widgets for terminal UIs
- [GoSNMP](https://github.com/gosnmp/gosnmp) - SNMP library for Go
