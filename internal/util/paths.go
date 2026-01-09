package util

import (
	"os"
	"path/filepath"
	"strings"
)

func DataDir(app string) string {
	if base := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); base != "" {
		return filepath.Join(base, app)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".", app)
	}
	return filepath.Join(home, ".local", "share", app)
}

func ReportsDir(app string) string {
	return filepath.Join(DocumentsDir(), strings.ToUpper(app))
}

func DocumentsDir() string {
	if base := strings.TrimSpace(os.Getenv("XDG_DOCUMENTS_DIR")); base != "" {
		return expandHome(base)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	configPath := filepath.Join(home, ".config", "user-dirs.dirs")
	if data, err := os.ReadFile(configPath); err == nil {
		if dir := parseUserDir(string(data), "XDG_DOCUMENTS_DIR"); dir != "" {
			return expandHome(dir)
		}
	}
	return filepath.Join(home, "Documents")
}

func parseUserDir(data, key string) string {
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, key+"=") {
			continue
		}
		value := strings.TrimPrefix(line, key+"=")
		value = strings.Trim(value, "\"")
		return value
	}
	return ""
}

func expandHome(path string) string {
	if !strings.Contains(path, "$HOME") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return strings.ReplaceAll(path, "$HOME", "")
	}
	return strings.ReplaceAll(path, "$HOME", home)
}
