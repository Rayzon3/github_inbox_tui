package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"golang.org/x/term"
)

const (
	keychainService = "github_inbox_tui"
	keychainAccount = "github"
	tokenFileName   = "token"
)

func loadToken() (string, error) {
	if runtime.GOOS == "darwin" {
		if token, err := loadTokenFromKeychain(); err == nil {
			return token, nil
		}
	}
	return loadTokenFromFile()
}

func saveToken(token string) error {
	if runtime.GOOS == "darwin" {
		if err := saveTokenToKeychain(token); err == nil {
			return nil
		}
	}
	return saveTokenToFile(token)
}

func promptToken() (string, error) {
	fmt.Fprint(os.Stderr, "Enter GITHUB_TOKEN: ")
	input, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("GITHUB_TOKEN is required")
	}
	token := strings.TrimSpace(string(input))
	if token == "" {
		return "", fmt.Errorf("GITHUB_TOKEN is required")
	}
	return token, nil
}

func loadTokenFromKeychain() (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-s", keychainService, "-a", keychainAccount, "-w")
	out, err := cmd.Output()
	if err != nil {
		return "", err		
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", errors.New("empty token in keychain")
	}
	return token, nil
}

func saveTokenToKeychain(token string) error {
	if token == "" {
		return errors.New("empty token")
	}
	cmd := exec.Command("security", "add-generic-password", "-s", keychainService, "-a", keychainAccount, "-w", token, "-U")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func tokenFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "github_inbox_tui")
	return filepath.Join(dir, tokenFileName), nil
}

func loadTokenFromFile() (string, error) {
	path, err := tokenFilePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", errors.New("empty token on disk")
	}
	return token, nil
}

func saveTokenToFile(token string) error {
	if token == "" {
		return errors.New("empty token")
	}
	path, err := tokenFilePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return err
	}
	return nil
}
