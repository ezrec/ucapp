#!/bin/sh

rm -rf depot/000000.ud
mkdir -p depot/000000.ud

go run ./cmd/ucapp build os/shell.uc
go run ./cmd/ucapp build os/list.uc
go run ./cmd/ucapp depot save LIST os/list.ur
go run ./cmd/ucapp depot save 0x00 os/shell.ur
go run ./cmd/ucapp depot list
