package live

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func s(cmd string) []string {
	return strings.Split(cmd, " ")
}

func sf(format string, args ...any) []string {
	return strings.Split(fmt.Sprintf(format, args...), " ")
}

func setDeployer(workDir, file, oldDeployer, newDeployer string) error {
	path := filepath.Join(workDir, "script", "deploy", file)
	script, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	newScript := strings.ReplaceAll(string(script), oldDeployer, newDeployer)
	return os.WriteFile(path, []byte(newScript), 0644)
}

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
