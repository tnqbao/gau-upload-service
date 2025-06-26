#!/bin/sh
# Start the application
echo "Starting API..."
if [ -f "gau-upload-service.bin" ]; then
    ./gau-upload-service.bin
else
    echo "Running main.go..."
    go run main.go
fi