package spool

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/spanner"
	admin "cloud.google.com/go/spanner/admin/database/apiv1"
	"github.com/cloudspannerecosystem/spool/model"
	databasepb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
)

func connect(ctx context.Context, t *testing.T, conf *Config) (*spanner.Client, func()) {
	t.Helper()
	client, err := spanner.NewClient(ctx, conf.Database(), conf.ClientOptions()...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	return client, func() {
		if err := clean(ctx, client, conf,
			func(ctx context.Context, txn *spanner.ReadWriteTransaction) ([]*model.SpoolDatabase, error) {
				return model.FindAllSpoolDatabases(ctx, txn)
			},
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()

	cfg := ConfigTestDatabase(t)

	ctx := context.Background()

	adminClient, err := admin.NewDatabaseAdminClient(ctx, cfg.ClientOptions()...)
	if err != nil {
		t.Fatal(err)
	}

	op, err := adminClient.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          cfg.Instance(),
		CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", cfg.DatabaseID()),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := op.Wait(ctx); err != nil {
		t.Fatal(err)
	}

	if err := Setup(ctx, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestListAll(t *testing.T) {
	t.Parallel()

	cfg := SetupTestDatabase(t)

	ctx := context.Background()
	client, truncate := connect(ctx, t, cfg)
	t.Cleanup(truncate)
	sdb1 := &model.SpoolDatabase{
		DatabaseName: "zoncoen-spool-test-1",
		Checksum:     "checksum-1",
		State:        StateIdle.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	sdb2 := &model.SpoolDatabase{
		DatabaseName: "zoncoen-spool-test-2",
		Checksum:     "checksum-2",
		State:        StateIdle.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	if _, err := client.Apply(ctx, []*spanner.Mutation{sdb1.Insert(ctx), sdb2.Insert(ctx)}); err != nil {
		t.Fatalf("failed to setup fixture: %s", err)
	}

	sdbs, err := ListAll(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(sdbs) != 2 {
		t.Errorf("failed to find all: found %d", len(sdbs))
	}
}

func TestCleanAll(t *testing.T) {
	t.Parallel()

	cfg := SetupTestDatabase(t)

	ctx := context.Background()
	client, truncate := connect(ctx, t, cfg)
	t.Cleanup(truncate)
	sdb1 := &model.SpoolDatabase{
		DatabaseName: "zoncoen-spool-test-1",
		Checksum:     "checksum-1",
		State:        StateIdle.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	sdb2 := &model.SpoolDatabase{
		DatabaseName: "zoncoen-spool-test-2",
		Checksum:     "checksum-2",
		State:        StateIdle.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	if _, err := client.Apply(ctx, []*spanner.Mutation{sdb1.Insert(ctx), sdb2.Insert(ctx)}); err != nil {
		t.Fatalf("failed to setup fixture: %s", err)
	}

	if err := CleanAll(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	sdbs, err := model.FindAllSpoolDatabases(ctx, client.Single())
	if err != nil {
		t.Error(err)
	}
	if len(sdbs) != 0 {
		t.Error("failed to clean all")
	}
}
