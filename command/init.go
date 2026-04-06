package command

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ttl-cli/i18n"

	"github.com/spf13/cobra"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: i18n.T("command.init.short"),
	Long:  i18n.T("command.init.long"),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func runInit() error {
	// 1. 检测 shell 类型
	shell := detectShell()
	Println(i18n.T("command.init.detected_shell", shell))

	if shell == "unknown" {
		Println(i18n.T("command.init.unsupported_shell"))
		Println()
		Println(i18n.T("command.init.manual_setup"))
		return nil
	}

	// 2. 创建补全脚本目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.New(i18n.T("command.init.error_home_dir", err))
	}

	completionDir := filepath.Join(homeDir, ".ttl", "completion")
	if err := os.MkdirAll(completionDir, 0755); err != nil {
		return errors.New(i18n.T("command.init.error_create_dir", err))
	}

	// 3. 生成补全脚本
	scriptPath, err := generateCompletionScript(shell, completionDir)
	if err != nil {
		return errors.New(i18n.T("command.init.error_generate", err))
	}

	Println(i18n.T("command.init.script_created", scriptPath))

	// 4. 配置 shellrc
	rcFile, err := setupShellRC(shell, scriptPath)
	if err != nil {
		return errors.New(i18n.T("command.init.error_setup_rc", err))
	}

	// 5. 输出结果
	Println()
	Println(i18n.T("command.init.success"))
	Println(i18n.T("command.init.script_path", scriptPath))

	if rcFile != "" {
		Println(i18n.T("command.init.rc_configured", rcFile))
		Println()
		Println(i18n.T("command.init.reload_hint"))
		Printf("  source %s\n", rcFile)
	} else {
		// rcFile 为空表示已配置，无需再次配置
		Println(i18n.T("command.init.already_configured"))
		Println()
	}

	return nil
}

// detectShell 检测当前 shell 类型
func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "unknown"
	}

	// 提取最后一部分: /bin/zsh -> zsh
	if idx := strings.LastIndex(shell, "/"); idx >= 0 {
		shell = shell[idx+1:]
	}

	// 标准化名称
	switch shell {
	case "zsh", "bash", "fish":
		return shell
	default:
		return "unknown"
	}
}

// generateCompletionScript 生成补全脚本
func generateCompletionScript(shell string, completionDir string) (string, error) {
	var scriptName string
	switch shell {
	case "zsh":
		scriptName = "_ttl"
	case "bash":
		scriptName = "ttl"
	case "fish":
		scriptName = "ttl.fish"
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}

	scriptPath := filepath.Join(completionDir, scriptName)

	// 调用 ttl completion <shell> 生成脚本
	cmd := exec.Command(os.Args[0], "__complete", shell)
	// 使用 cobra 内置的 completion 命令
	cmd = exec.Command(os.Args[0], "completion", shell)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(scriptPath, output, 0644); err != nil {
		return "", err
	}

	return scriptPath, nil
}

// setupShellRC 配置 shellrc
func setupShellRC(shell string, scriptPath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var rcFile string
	switch shell {
	case "zsh":
		rcFile = filepath.Join(homeDir, ".zshrc")
	case "bash":
		// 优先使用 .bashrc，如果不存在则使用 .bash_profile
		bashrc := filepath.Join(homeDir, ".bashrc")
		bashProfile := filepath.Join(homeDir, ".bash_profile")
		if _, err := os.Stat(bashrc); err == nil {
			rcFile = bashrc
		} else {
			rcFile = bashProfile
		}
	case "fish":
		rcFile = filepath.Join(homeDir, ".config", "fish", "config.fish")
	default:
		return "", nil
	}

	// 检测标记
	markerStart := "# >>> ttl completion >>>"
	markerEnd := "# <<< ttl completion <<<"

	// 读取现有内容
	content, err := os.ReadFile(rcFile)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	// 检查是否已配置
	if strings.Contains(string(content), markerStart) {
		// 已配置，返回空字符串表示跳过
		return "", nil
	}

	// 创建目录（针对 fish）
	rcDir := filepath.Dir(rcFile)
	if err := os.MkdirAll(rcDir, 0755); err != nil {
		return "", err
	}

	// 追加配置
	var sourceLine string
	if shell == "fish" {
		sourceLine = fmt.Sprintf("\n%s\nsource %s\n%s\n", markerStart, scriptPath, markerEnd)
	} else {
		sourceLine = fmt.Sprintf("\n%s\nsource %s\n%s\n", markerStart, scriptPath, markerEnd)
	}

	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.WriteString(sourceLine); err != nil {
		return "", err
	}

	return rcFile, nil
}
