package config

import (
	"os"
	"testing"
)

func TestLoadRejectsWeakAdminPass(t *testing.T) {
	weak := []string{"admin", "changeme", "password", ""}
	for _, p := range weak {
		t.Run("weak="+p, func(t *testing.T) {
			os.Setenv("ADMIN_PASS", p)
			os.Unsetenv("ALLOW_WEAK_AUTH")
			defer os.Unsetenv("ADMIN_PASS")
			_, err := Load()
			if err == nil {
				t.Fatalf("expected error for weak password %q", p)
			}
		})
	}
}

func TestLoadAcceptsStrongPass(t *testing.T) {
	os.Setenv("ADMIN_PASS", "Tr0ub4dor&3-very-strong")
	defer os.Unsetenv("ADMIN_PASS")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.AdminPass == "" {
		t.Fatal("expected admin pass set")
	}
}

func TestLoadAcceptsWeakWithFlag(t *testing.T) {
	os.Setenv("ADMIN_PASS", "admin")
	os.Setenv("ALLOW_WEAK_AUTH", "1")
	defer os.Unsetenv("ADMIN_PASS")
	defer os.Unsetenv("ALLOW_WEAK_AUTH")
	if _, err := Load(); err != nil {
		t.Fatalf("expected ok with ALLOW_WEAK_AUTH=1, got %v", err)
	}
}
