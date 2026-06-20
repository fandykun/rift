package db

import (
	"context"
	"testing"

	"github.com/fandykun/rift/internal/config"
)

func TestNewPoolRequiresConfig(t *testing.T) {
	_, err := NewPool(context.Background(), nil)
	if err == nil {
		t.Fatal("NewPool returned nil error for nil config")
	}
}

func TestNewPoolRequiresDatabaseURL(t *testing.T) {
	cfg := config.Default()

	_, err := NewPool(context.Background(), &cfg)
	if err == nil {
		t.Fatal("NewPool returned nil error for empty DatabaseURL")
	}
}

func TestPingRejectsNilPool(t *testing.T) {
	if err := Ping(context.Background(), nil); err == nil {
		t.Fatal("Ping returned nil error for nil pool")
	}
}
