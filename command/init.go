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
	shell := detectShell()
	Println(i18n.T("command.init.detected_shell", shell))

	if shell == "unknown" {
		Println(i18n.T("command.init.unsupported_shell"))
		Println()
		Println(i18n.T("command.init.manual_setup"))
		return nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.New(i18n.T("command.init.error_home_dir", err))
	}

	completionDir := filepath.Join(homeDir, ".ttl", "completion")
	if err := os.MkdirAll(completionDir, 0755); err != nil {
		return errors.New(i18n.T("command.init.error_create_dir", err))
	}

	scriptPath, err := generateCompletionScript(shell, completionDir)
	if err != nil {
		return errors.New(i18n.T("command.init.error_generate", err))
	}

	Println(i18n.T("command.init.script_created", scriptPath))

	rcFile, err := setupShellRC(shell, scriptPath)
	if err != nil {
		return errors.New(i18n.T("command.init.error_setup_rc", err))
	}

	Println()
	Println(i18n.T("command.init.success"))
	Println(i18n.T("command.init.script_path", scriptPath))

	if rcFile != "" {
		Println(i18n.T("command.init.rc_configured", rcFile))
		Println()
		Println(i18n.T("command.init.reload_hint"))
		Printf("  source %s\n", rcFile)
	} else {
		Println(i18n.T("command.init.already_configured"))
		Println()
	}

	return nil
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "unknown"
	}

	if idx := strings.LastIndex(shell, "/"); idx >= 0 {
		shell = shell[idx+1:]
	}

	switch shell {
	case "zsh", "bash", "fish":
		return shell
	default:
		return "unknown"
	}
}

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

	cmd := exec.Command(os.Args[0], "__complete", shell)
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

	markerStart := "# >>> ttl completion >>>"
	markerEnd := "# <<< ttl completion <<<"

	content, err := os.ReadFile(rcFile)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if strings.Contains(string(content), markerStart) {
		return "", nil
	}

	rcDir := filepath.Dir(rcFile)
	if err := os.MkdirAll(rcDir, 0755); err != nil {
		return "", err
	}

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
