# ElastiFlow SNMP Tester

A Terminal User Interface (TUI) application for testing ElastiFlow SNMP configurations.

## Overview

ElastiFlow SNMP Tester is a Go-based TUI application that helps you test and troubleshoot your ElastiFlow SNMP collector configurations. Unlike the standard ElastiFlow SNMP collector, which requires you to start the full collector and sift through log files, this tool provides immediate feedback on your SNMP connectivity and data collection.

It supports testing SNMP connectivity to all configured devices and performing detailed SNMP GET/WALK operations for specific device groups against target devices.

## Features

- **Connectivity Testing:** Test basic SNMP connectivity (sysObjectID GET) to all configured devices in parallel.
- **Device Group Testing:**
    - Test specific device groups against individual devices.
    - Test all groups for all configured devices, or only for currently displayed/filtered devices.
    - Device tests run in parallel (configurable concurrency for OID polling per device, and a separate limit for concurrent device tests).
- **Detailed Result View:** Inspect individual group test results, including the exact OID instances and values retrieved or error messages per OID.
- **SNMP Support:** SNMPv2c and SNMPv3.
- **User Interface:** Easy-to-use terminal interface with real-time feedback.
- **Device Search:** Filter the device list by name, IP, or source file.
- **CSV Export:** Export connectivity test results and "All Devices/Groups" test results to CSV files.

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
    - Displays SNMP test results for the selected device and group. OIDs within the group are polled in parallel, respecting the device's `MaxConcurrentPolls` setting.
    - **Key Bindings:**
        - `ESC`: Return to the group selection screen for the current device.
        - `r`: Restart the test for this specific device and group.

4.  **All Devices/Groups Results Screen (after pressing 'a' or 's' on Device Selection):**
    - Displays results for multiple device-group tests. Devices are tested in parallel (up to `maxConcurrentDeviceTests`), and OIDs within each group are also polled in parallel.
    - The table updates incrementally as results come in.
    - **Key Bindings:**
        - `↑`/`↓`: Navigate the results table.
        - `Enter`: On a selected result row, open the **Detailed Result View** for that specific device-group test.
        - `ESC`: Return to the device selection screen.
        - `e`: Export all displayed results to a CSV file.

5.  **Detailed Result View (after pressing Enter on a result in "All Devices/Groups Results"):**
    - Shows detailed information for a single device-group test, including:
        - Device Name, IP, Group Name, Status, OIDs Retrieved, Duration, Error Message.
        - **Walked OID Data:** A list of OID names from the group, and for each:
            - Any error encountered while walking that specific OID.
            - "No data retrieved" if the walk was successful but returned no values.
            - A list of `instance = value` pairs for all data points retrieved under that OID.
    - **Key Bindings:**
        - `ESC`: Return to the "All Devices/Groups Results" screen.

## Troubleshooting

If you encounter any issues:

1.  Ensure the ElastiFlow SNMP configuration directory (e.g., `/etc/elastiflow/snmp` or your custom path) exists and contains valid `device.collection-*.yml` and `oid.collection-*.yml` files.
2.  Verify your SNMP device configurations within the YAML files (IP addresses, communities, SNMPv3 credentials, port numbers, `MaxConcurrentPolls`).
3.  Make sure your network allows SNMP traffic (typically UDP port 161) to the target devices from the machine running `snmptester`.
4.  Check for typos in device names or group names referenced in your configurations.

## License

MIT License

## Acknowledgements

- [ElastiFlow](https://github.com/robcowart/elastiflow) - Network flow monitoring and analytics solution
- [tcell](https://github.com/gdamore/tcell) - Terminal handling library
- [tview](https://github.com/rivo/tview) - Rich interactive widgets for terminal UIs
- [GoSNMP](https://github.com/gosnmp/gosnmp) - SNMP library for Go

- ![image](https://github.com/user-attachments/assets/28004a7a-406c-454a-b4ed-b87359ca6ba0)
- ![image](https://github.com/user-attachments/assets/b4b35648-e228-440f-91dd-14be25c341af)
- ![image](https://github.com/user-attachments/assets/8455a8f4-b315-46b6-8ba2-60c821a1de2a)
- ![image](https://github.com/user-attachments/assets/26fcb352-aff7-4001-ab1b-57baa35034d9)

- 

- 


