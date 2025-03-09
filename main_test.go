package spool

import (
	"context"
	"crypto/rand"
	_ "embed"
	"fmt"
	"os"
	"testing"
)

var (
	//go:embed testdata/schema1.sql
	ddl1 []byte
	//go:embed testdata/schema2.sql
	ddl2 []byte
)

func spoolSpannerDatabaseNamePrefix() string {
	s := os.Getenv("SPOOL_SPANNER_DATABASE_NAME_PREFIX")
	if s == "" {
		s = "spool-test"
	}
	return s
}

func SetupTestDatabase(t *testing.T) *Config {
	t.Helper()

	emulatorHost := os.Getenv("SPANNER_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Fatal("SPANNER_EMULATOR_HOST environment variable is not set")
	}

	ctx := context.Background()

	b := make([]byte, 10)
	_, err := rand.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig(
		os.Getenv("SPANNER_PROJECT_ID"),
		os.Getenv("SPANNER_INSTANCE_ID"),
		fmt.Sprintf("test-%x", b),
	)

	err = Setup(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	return cfg
}
