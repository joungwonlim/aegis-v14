// Package cmd - quant CLI commands
package cmd

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	// 공통 플래그
	cfgFile string
	verbose bool
)

// rootCmd 루트 커맨드
var rootCmd = &cobra.Command{
	Use:   "quant",
	Short: "Aegis v14 Quant Trading System - CLI",
	Long: `Aegis v14 Quant Trading System - CLI

Usage:
    go run ./cmd/quant [command]

Commands:
    backend     start/stop    - Backend Runtime Server (Port 8099)
    frontend    start         - Frontend Dev Server (Port 3000)
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
}

// Execute 루트 커맨드 실행
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .env)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(backendCmd)
	rootCmd.AddCommand(frontendCmd)
}

// initConfig reads in config file and ENV variables if set
func initConfig() error {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		// .env 파일이 없어도 계속 진행 (환경변수로 설정 가능)
		if verbose {
			fmt.Println("Warning: .env file not found, using environment variables")
		}
	}

	return nil
}
