#!/bin/sh

# Remove the existing drum 000000
rm -rf depot/000000.ud
mkdir -p depot/000000.ud

# Build the drum 000000 tools
go run ./cmd/ucapp build os/shell.uc
go run ./cmd/ucapp build os/list.uc

# Write the shell to the boot ring.
go run ./cmd/ucapp depot save 0x00 os/shell.ur

# Save the rest of the tools
go run ./cmd/ucapp depot save LIST os/list.ur

# List the depot to the user
go run ./cmd/ucapp depot list
