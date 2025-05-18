#!/bin/bash

# Navigate to the project directory
cd "$(dirname "$0")"

# Run go mod tidy to ensure all dependencies are correct
echo "Running go mod tidy..."
go mod tidy

# Clean any previous build
rm -f snmptester

# Build the application
echo "Building ElastiFlow SNMP Tester..."
go build -o snmptester ./cmd/snmptester

# Check if build was successful
if [ $? -eq 0 ]; then
    echo "Build successful!"
    chmod +x snmptester
    echo "Usage: ./snmptester [options]"
    echo "Options:"
    echo "  -c, --config string   Path to ElastiFlow SNMP configuration directory (default: \"/etc/elastiflow/snmp\")"
    echo "  -h, --help            Show help"
    echo "  -v, --version         Show version"
    echo ""
    echo "Example: ./snmptester -c /home/hotwirez/elastiflow_snmp_test/example/elastiflow/snmp"
else
    echo "Build failed. Please check the errors above."
fi
