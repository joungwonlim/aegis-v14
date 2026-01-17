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

	// Monitor both processes
	errCh := make(chan error, 2)
	go func() {
		errCh <- runtimeCmd.Wait()
	}()
	go func() {
		errCh <- apiCmd.Wait()
	}()

	fmt.Println("✅ Backend 서버 실행 중")
	fmt.Println("   - Runtime 서버: Exit Engine, Price Sync")
	fmt.Println("   - API 서버: http://localhost:8099")
	fmt.Println("종료하려면 Ctrl+C를 누르세요")

	// Wait for signal or process exit
	select {
	case <-sigCh:
		fmt.Println("\n🛑 종료 신호 수신, 서버 종료 중...")
	case err := <-errCh:
		if err != nil {
			fmt.Printf("\n⚠️  프로세스가 예기치 않게 종료됨: %v\n", err)
		}
	}

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
	killed := false

	// 1. 포트 기반 종료 (API 서버: 8099)
	if killProcessByPort("8099") {
		killed = true
	}

	// 2. pkill로 강제 종료 (go run 임시 바이너리 + 서비스명)
	pkillPatterns := []string{
		"exe/runtime",  // go run으로 실행된 runtime
		"exe/api",      // go run으로 실행된 api
		"aegis-v14-runtime",
		"aegis-v14-api",
	}
	for _, pattern := range pkillPatterns {
		if pkillProcess(pattern) {
			killed = true
		}
	}

	// 3. 프로세스 패턴 기반 종료 (go run 명령어 자체)
	pgrepPatterns := []string{"cmd/runtime", "cmd/api"}
	for _, pattern := range pgrepPatterns {
		if killProcessByPattern(pattern) {
			killed = true
		}
	}

	if killed {
		// 포트가 해제될 때까지 대기
		exec.Command("sleep", "1").Run()
	}
}

// pkillProcess pkill로 프로세스 강제 종료
func pkillProcess(pattern string) bool {
	// 먼저 프로세스 존재 여부 확인
	check := exec.Command("pgrep", "-f", pattern)
	if err := check.Run(); err != nil {
		return false // 프로세스 없음
	}

	fmt.Printf("pkill로 프로세스 종료: %s\n", pattern)

	// SIGTERM 시도
	exec.Command("pkill", "-f", pattern).Run()

	// 잠시 대기 후 SIGKILL
	exec.Command("sleep", "0.5").Run()
	exec.Command("pkill", "-9", "-f", pattern).Run()

	return true
}

// killProcessByPort 지정된 포트를 사용하는 프로세스 종료
func killProcessByPort(port string) bool {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%s", port)).Output()
	if err != nil {
		return false
	}

	pids := strings.Fields(strings.TrimSpace(string(out)))
	if len(pids) == 0 {
		return false
	}

	fmt.Printf("포트 %s 사용 프로세스 종료 중... (PIDs: %v)\n", port, pids)

	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		if proc, err := os.FindProcess(pid); err == nil {
			proc.Signal(syscall.SIGTERM)
			// 잠시 대기 후 강제 종료
			go func(p *os.Process) {
				exec.Command("sleep", "0.5").Run()
				p.Signal(syscall.SIGKILL)
			}(proc)
		}
	}
	return true
}

// killProcessByPattern 패턴과 일치하는 프로세스 종료
func killProcessByPattern(pattern string) bool {
	cmd := exec.Command("pgrep", "-f", pattern)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return false
	}

	pids := strings.TrimSpace(out.String())
	if pids == "" {
		return false
	}

	killed := false
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

		if proc, err := os.FindProcess(pid); err == nil {
			fmt.Printf("프로세스 종료: %s (PID: %d)\n", pattern, pid)
			proc.Signal(syscall.SIGTERM)
			go func(p *os.Process) {
				exec.Command("sleep", "0.5").Run()
				p.Signal(syscall.SIGKILL)
			}(proc)
			killed = true
		}
	}
	return killed
}
