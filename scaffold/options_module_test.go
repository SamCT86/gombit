package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateModuleUsesGoModulePathRules(t *testing.T) {
	for _, modulePath := range []string{
		"example.com/demo@v2",
		"example.com/demo;x",
		"example.com/demo\tbad",
	} {
		t.Run(modulePath, func(t *testing.T) {
			if err := validateModule(modulePath); err == nil {
				t.Fatalf("validateModule(%q) error = nil, want Go module-path rejection", modulePath)
			}
		})
	}

	for _, modulePath := range []string{
		"example.com/demo",
		"example.com/demo/v2",
	} {
		t.Run(modulePath, func(t *testing.T) {
			if err := validateModule(modulePath); err != nil {
				t.Fatalf("validateModule(%q) error = %v, want nil", modulePath, err)
			}
		})
	}
}

func TestGenerateRejectsMalformedModuleBeforeWritingDestination(t *testing.T) {
	workDir := t.TempDir()
	dest := filepath.Join(workDir, "demo")

	err := Generate(context.Background(), Options{
		Name:     "demo",
		Module:   "example.com/demo@v2",
		Database: "sqlite",
		WorkDir:  workDir,
	})
	if err == nil || !strings.Contains(err.Error(), "invalid module path") {
		t.Fatalf("Generate() error = %v, want invalid module path", err)
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Fatalf("destination exists after validation failure: stat error = %v", statErr)
	}
}
