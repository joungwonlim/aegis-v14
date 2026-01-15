package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// backendCmd backend 서브커맨드
var backendCmd = &cobra.Command{
	Use:   "backend",
	Short: "Backend 서버 관리",
	Long: `Backend 서버를 실행합니다 (Exit Engine, Price Sync, API 서버).

Examples:
  go run ./cmd/quant backend start    # Backend 서버 시작 (Runtime + API)
  go run ./cmd/quant backend stop     # Backend 서버 종료`,
}

// backendStartCmd backend 서버 시작
var backendStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Backend 서버 시작",
	Long:  `Backend 서버를 시작합니다 (Exit Engine + Price Sync + API). Ctrl+C로 종료할 수 있습니다.`,
	RunE:  runBackendStart,
}

// backendStopCmd backend 서버 종료
var backendStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Backend 서버 종료",
	Long:  `실행 중인 Backend 서버를 종료합니다 (Runtime + API).`,
	RunE:  runBackendStop,
}

func init() {
	backendCmd.AddCommand(backendStartCmd)
	backendCmd.AddCommand(backendStopCmd)
}

func runBackendStart(cmd *cobra.Command, args []string) error {
	// 기존 프로세스 종료
	killExistingBackend()

	fmt.Println("🚀 Backend 서버 시작...")

	// 1. Run cmd/runtime (Exit Engine + Price Sync)
	runtimeCmd := exec.Command("go", "run", "./cmd/runtime")
	runtimeCmd.Stdout = os.Stdout
	runtimeCmd.Stderr = os.Stderr
	runtimeCmd.Env = os.Environ()

	if err := runtimeCmd.Start(); err != nil {
		return fmt.Errorf("failed to start runtime: %w", err)
	}

	// 2. Run cmd/api (API Server)
	apiCmd := exec.Command("go", "run", "./cmd/api")
	apiCmd.Stdout = os.Stdout
	apiCmd.Stderr = os.Stderr
	apiCmd.Env = os.Environ()

	if err := apiCmd.Start(); err != nil {
		// Runtime 종료 후 에러 반환
		runtimeCmd.Process.Kill()
		return fmt.Errorf("failed to start API server: %w", err)
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("✅ Backend 서버 실행 중")
	fmt.Println("   - Exit Engine 평가 루프 (3초 주기)")
	fmt.Println("   - Price Sync 서비스 (3초 주기)")
	fmt.Println("   - API 서버 (포트: 8099)")
	fmt.Println("종료하려면 Ctrl+C를 누르세요")

	<-sigCh
	fmt.Println("\n🛑 종료 신호 수신, 서버 종료 중...")

	// 두 프로세스 모두 종료
	if err := runtimeCmd.Process.Kill(); err != nil {
		fmt.Printf("Runtime 종료 실패: %v\n", err)
	}
	if err := apiCmd.Process.Kill(); err != nil {
		fmt.Printf("API 서버 종료 실패: %v\n", err)
	}

	fmt.Println("✅ Backend 서버 종료 완료")
	return nil
}

func runBackendStop(cmd *cobra.Command, args []string) error {
	fmt.Println("🛑 Backend 서버 종료 중...")

	// 기존 프로세스 종료
	killExistingBackend()

	fmt.Println("✅ Backend 서버 종료 완료")
	return nil
}

// killExistingBackend 기존 백엔드 프로세스 종료
func killExistingBackend() {
	// pgrep으로 기존 프로세스 찾기
	patterns := []string{"cmd/runtime", "cmd/api", "quant backend start"}

	for _, pattern := range patterns {
		cmd := exec.Command("pgrep", "-f", pattern)
		var out bytes.Buffer
		cmd.Stdout = &out

		if err := cmd.Run(); err != nil {
			continue // 프로세스 없음
		}

		pids := strings.TrimSpace(out.String())
		if pids == "" {
			continue
		}

		// 현재 프로세스 PID 제외
		currentPID := os.Getpid()
		for _, pidStr := range strings.Split(pids, "\n") {
			pidStr = strings.TrimSpace(pidStr)
			if pidStr == "" {
				continue
			}

			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				continue
			}

			// 현재 프로세스와 부모 프로세스 제외
			if pid == currentPID || pid == os.Getppid() {
				continue
			}

			// 프로세스 종료 (SIGTERM)
			if proc, err := os.FindProcess(pid); err == nil {
				fmt.Printf("기존 백엔드 프로세스 종료 (PID: %d)\n", pid)
				proc.Signal(syscall.SIGTERM)

				// 잠시 대기 후 강제 종료
				go func(p *os.Process) {
					exec.Command("sleep", "0.1").Run()
					p.Signal(syscall.SIGKILL)
				}(proc)
			}
		}
	}

	// 포트가 해제될 때까지 잠시 대기
	exec.Command("sleep", "0.5").Run()
}
