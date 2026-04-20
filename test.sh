#!/bin/sh
cd "$(dirname "$0")" || exit 1
go test -v .
go test -race .
