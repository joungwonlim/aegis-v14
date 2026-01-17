package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var frontendPort string

// frontendCmd frontend 서브커맨드
var frontendCmd = &cobra.Command{
	Use:   "frontend",
	Short: "Frontend 개발 서버 관리",
	Long: `Frontend 개발 서버를 실행합니다 (Next.js App Router).

Examples:
  go run ./cmd/quant frontend start              # Dev 서버 시작 (기본: 3099)
  go run ./cmd/quant frontend start --port=3001  # 포트 지정
  go run ./cmd/quant frontend stop               # Dev 서버 종료 (기본: 3099)
  go run ./cmd/quant frontend stop --port=3001   # 특정 포트 종료`,
}

// frontendStartCmd frontend 서버 시작
var frontendStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Frontend 개발 서버 시작",
	Long:  `Next.js 개발 서버를 시작합니다. Ctrl+C로 종료할 수 있습니다.`,
	RunE:  runFrontendStart,
}

// frontendStopCmd frontend 서버 종료
var frontendStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Frontend 개발 서버 종료",
	Long:  `실행 중인 Frontend 개발 서버를 종료합니다 (포트 기반).`,
	RunE:  runFrontendStop,
}

func init() {
	frontendStartCmd.Flags().StringVar(&frontendPort, "port", "3099", "Dev 서버 포트")
	frontendStopCmd.Flags().StringVar(&frontendPort, "port", "3099", "종료할 서버 포트")
	frontendCmd.AddCommand(frontendStartCmd)
	frontendCmd.AddCommand(frontendStopCmd)
}

func runFrontendStart(cmd *cobra.Command, args []string) error {
	// 기존 프로세스 종료
	killExistingFrontend(frontendPort)

	fmt.Printf("🚀 Frontend 개발 서버 시작 (포트: %s)...\n", frontendPort)

	// Find frontend directory (v14/frontend from v14/backend)
	frontendDir := filepath.Join("..", "frontend")
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		return fmt.Errorf("frontend directory not found: %s", frontendDir)
	}

	// Check if node_modules exists
	nodeModules := filepath.Join(frontendDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		fmt.Println("⚠️  node_modules not found, running 'pnpm install' first...")
		installCmd := exec.Command("pnpm", "install")
		installCmd.Dir = frontendDir
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to run 'pnpm install': %w", err)
		}
	}

	// Run pnpm dev
	devCmd := exec.Command("pnpm", "dev", "--port", frontendPort)
	devCmd.Dir = frontendDir
	devCmd.Stdout = os.Stdout
	devCmd.Stderr = os.Stderr
	devCmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%s", frontendPort))

	// Start the dev server
	if err := devCmd.Start(); err != nil {
		return fmt.Errorf("failed to start frontend server: %w", err)
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("✅ Frontend 개발 서버 실행 중\n")
	fmt.Printf("   - URL: http://localhost:%s\n", frontendPort)
	fmt.Println("종료하려면 Ctrl+C를 누르세요")

	<-sigCh
	fmt.Println("\n🛑 종료 신호 수신, 서버 종료 중...")

	if err := devCmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill frontend server: %w", err)
	}

	fmt.Println("✅ Frontend 개발 서버 종료 완료")
	return nil
}

func runFrontendStop(cmd *cobra.Command, args []string) error {
	fmt.Printf("🛑 Frontend 서버 종료 중 (포트: %s)...\n", frontendPort)

	// 포트 기반으로 프로세스 종료
	killExistingFrontend(frontendPort)

	fmt.Println("✅ Frontend 서버 종료 완료")
	return nil
}

// killExistingFrontend 지정된 포트에서 실행 중인 프로세스를 종료합니다.
func killExistingFrontend(port string) {
	// lsof로 해당 포트를 사용하는 프로세스 찾기
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%s", port)).Output()
	if err != nil {
		// 프로세스가 없으면 에러 발생 (정상)
		return
	}

	pids := strings.Fields(strings.TrimSpace(string(out)))
	if len(pids) == 0 {
		return
	}

	fmt.Printf("기존 프로세스 종료 중... (포트: %s, PIDs: %v)\n", port, pids)

	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		// SIGKILL로 강제 종료
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Kill()
		}
	}

	// 프로세스 종료 대기
	exec.Command("sleep", "1").Run()
}
