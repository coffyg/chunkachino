#!/bin/sh
cd "$(dirname "$0")" || exit 1
go test -bench=BenchmarkChunker -run=^$ -benchmem -benchtime=5s
