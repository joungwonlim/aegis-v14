// Package main - quant CLI
// 통합 CLI 진입점
//
// 사용법:
//
//	go run ./cmd/quant backend start
//	go run ./cmd/quant frontend start
package main

import (
	"os"

	"github.com/wonny/aegis/v14/cmd/quant/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
