package tui

import (
	"os"
	"path/filepath"
	"strings"
)

const recentMax = 20

func recentPath() (string, error) {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "bass2tably", "recent"), nil
}

func loadRecent() []string {
	p, err := recentPath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if fi, err := os.Stat(line); err != nil || fi.IsDir() {
			continue
		}
		out = append(out, line)
	}
	return out
}

func pushRecent(path string) {
	p, err := recentPath()
	if err != nil {
		return
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}
	list := []string{abs}
	for _, r := range loadRecent() {
		if r != abs && len(list) < recentMax {
			list = append(list, r)
		}
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	os.WriteFile(p, []byte(strings.Join(list, "\n")+"\n"), 0o644)
}
