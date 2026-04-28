package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mystravastats/internal/platform/runtimeconfig"
)

const (
	defaultOSRMControlComposeFile = "docker-compose-routing-osrm.yml"
	defaultOSRMControlTimeoutMs   = 30000
	maxOSRMControlOutputChars     = 4000
)

type OSRMControlAdapter struct {
	timeout time.Duration
}

type OSRMControlResult struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	Command     string `json:"command"`
	ProjectDir  string `json:"projectDir"`
	ComposeFile string `json:"composeFile"`
	Output      string `json:"output,omitempty"`
}

type OSRMControlError struct {
	StatusCode  int
	Description string
}

func (err OSRMControlError) Error() string {
	return err.Description
}

func NewOSRMControlAdapter() *OSRMControlAdapter {
	return &OSRMControlAdapter{
		timeout: time.Duration(osrmControlTimeoutMs()) * time.Millisecond,
	}
}

func (adapter *OSRMControlAdapter) StartOSRM(ctx context.Context) (OSRMControlResult, error) {
	projectDir, composeFile, composeExists := resolveOSRMControlComposeFile()
	dockerBin, dockerAvailable := resolveDockerBinary()
	command := []string{dockerBin, "compose", "-f", composeFile, "up", "-d", "osrm"}
	result := OSRMControlResult{
		Status:      "unavailable",
		Message:     "OSRM start command was not run.",
		Command:     commandDisplay(command),
		ProjectDir:  projectDir,
		ComposeFile: composeFile,
	}

	if !runtimeconfig.BoolValue("OSRM_CONTROL_ENABLED", true) {
		return result, OSRMControlError{
			StatusCode:  http.StatusForbidden,
			Description: "OSRM control is disabled. Set OSRM_CONTROL_ENABLED=true to allow starting OSRM from the UI.",
		}
	}
	if !composeExists {
		return result, OSRMControlError{
			StatusCode:  http.StatusConflict,
			Description: fmt.Sprintf("OSRM compose file not found: %s", composeFile),
		}
	}
	if !dockerAvailable {
		return result, OSRMControlError{
			StatusCode:  http.StatusConflict,
			Description: "Docker CLI not found. Install Docker Desktop or set OSRM_CONTROL_DOCKER_BIN.",
		}
	}

	startCtx, cancel := context.WithTimeout(ctx, adapter.timeout)
	defer cancel()

	cmd := exec.CommandContext(startCtx, dockerBin, "compose", "-f", composeFile, "up", "-d", "osrm")
	cmd.Dir = projectDir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	result.Output = trimOSRMControlOutput(output.String())
	if startCtx.Err() == context.DeadlineExceeded {
		return result, OSRMControlError{
			StatusCode:  http.StatusGatewayTimeout,
			Description: fmt.Sprintf("OSRM start command timed out after %s.", adapter.timeout),
		}
	}
	if err != nil {
		description := fmt.Sprintf("OSRM start command failed: %v", err)
		if result.Output != "" {
			description = fmt.Sprintf("%s. Output: %s", description, result.Output)
		}
		return result, OSRMControlError{
			StatusCode:  http.StatusInternalServerError,
			Description: description,
		}
	}

	result.Status = "started"
	result.Message = "OSRM start requested."
	return result, nil
}

func osrmControlTimeoutMs() int {
	timeoutMs := runtimeconfig.IntValue("OSRM_CONTROL_TIMEOUT_MS", defaultOSRMControlTimeoutMs)
	if timeoutMs < 1000 {
		return defaultOSRMControlTimeoutMs
	}
	return timeoutMs
}

func resolveOSRMControlComposeFile() (string, string, bool) {
	configuredProjectDir := strings.TrimSpace(runtimeconfig.StringValue("OSRM_CONTROL_PROJECT_DIR", ""))
	configuredComposeFile := strings.TrimSpace(runtimeconfig.StringValue("OSRM_CONTROL_COMPOSE_FILE", defaultOSRMControlComposeFile))
	if configuredComposeFile == "" {
		configuredComposeFile = defaultOSRMControlComposeFile
	}

	projectDir := configuredProjectDir
	if projectDir == "" {
		projectDir = discoverProjectDir()
	}
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}
	projectDir, _ = filepath.Abs(projectDir)

	composeFile := configuredComposeFile
	if filepath.IsAbs(composeFile) {
		if configuredProjectDir == "" {
			projectDir = filepath.Dir(composeFile)
		}
	} else {
		composeFile = filepath.Join(projectDir, composeFile)
	}
	composeFile, _ = filepath.Abs(composeFile)
	return projectDir, composeFile, fileExists(composeFile)
}

func discoverProjectDir() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return ""
	}
	current, err := filepath.Abs(workingDir)
	if err != nil {
		return workingDir
	}
	for {
		if fileExists(filepath.Join(current, defaultOSRMControlComposeFile)) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return workingDir
		}
		current = parent
	}
}

func resolveDockerBinary() (string, bool) {
	if configured := strings.TrimSpace(runtimeconfig.StringValue("OSRM_CONTROL_DOCKER_BIN", "")); configured != "" {
		if !strings.ContainsRune(configured, os.PathSeparator) {
			resolved, err := exec.LookPath(configured)
			if err == nil {
				return resolved, true
			}
			return configured, false
		}
		return configured, fileExists(configured)
	}
	if resolved, err := exec.LookPath("docker"); err == nil {
		return resolved, true
	}
	for _, candidate := range []string{"/usr/local/bin/docker", "/opt/homebrew/bin/docker"} {
		if fileExists(candidate) {
			return candidate, true
		}
	}
	return "docker", false
}

func commandDisplay(parts []string) string {
	displayParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.ContainsAny(part, " \t\n'\"") {
			displayParts = append(displayParts, "'"+strings.ReplaceAll(part, "'", "'\\''")+"'")
			continue
		}
		displayParts = append(displayParts, part)
	}
	return strings.Join(displayParts, " ")
}

func trimOSRMControlOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if len(trimmed) <= maxOSRMControlOutputChars {
		return trimmed
	}
	return trimmed[len(trimmed)-maxOSRMControlOutputChars:]
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
