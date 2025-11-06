package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const credFileName = "credentials.json"

type TokenInfo struct {
	Token     string     `json:"token"`
	Source    string     `json:"source"`     // "env" | "file"
	CreatedAt time.Time  `json:"created_at"` // when we saved to file
	ExpiresAt *time.Time `json:"expires_at"` // optional (JWT or server-provided)
}

func credsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home: %w", err)
	}
	return filepath.Join(home, ".tada"), nil
}

func credFilePath() (string, error) {
	dir, err := credsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, credFileName), nil
}

func GetToken() (*TokenInfo, error) {
	// 1) env override
	env := strings.TrimSpace(os.Getenv("TADA_TOKEN"))
	if env != "" {
		return &TokenInfo{Token: stripBearer(env), Source: "env"}, nil
	}

	// 2) file
	p, err := credFilePath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil // not logged in
		}
		return nil, fmt.Errorf("read credentials: %w", err)
	}
	var ti TokenInfo
	if err := json.Unmarshal(b, &ti); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	ti.Token = stripBearer(ti.Token)
	return &ti, nil
}

func SetToken(token string, expires *time.Time) error {
	token = stripBearer(strings.TrimSpace(token))
	if token == "" {
		return fmt.Errorf("empty token")
	}
	dir, err := credsDir()
	if err != nil {
		return err
	}
	// ensure ~/.tada exists with 0700
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	ti := TokenInfo{
		Token:     token,
		Source:    "file",
		CreatedAt: time.Now(),
		ExpiresAt: expires,
	}
	b, err := json.MarshalIndent(ti, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	p, _ := credFilePath()
	// write with 0600 (owner-only)
	if err := os.WriteFile(p, b, 0o600); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

func DeleteToken() error {
	p, err := credFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}

func stripBearer(s string) string {
	if strings.HasPrefix(strings.ToLower(s), "bearer ") {
		return strings.TrimSpace(s[7:])
	}
	return s
}
