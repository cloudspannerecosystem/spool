package spool

import (
	"context"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/gcpug/spool/model"
)

func connect(ctx context.Context, t *testing.T, conf *Config) (*spanner.Client, func()) {
	t.Helper()
	client, err := spanner.NewClient(ctx, conf.Database(), conf.ClientOptions()...)
	if err != nil {
		t.Fatal(err)
	}
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

func TestListAll(t *testing.T) {
	ctx := context.Background()
	client, truncate := connect(ctx, t, testConf.Config())
	defer truncate()
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

	sdbs, err := ListAll(ctx, testConf.Config())
	if err != nil {
		t.Fatal(err)
	}
	if len(sdbs) != 2 {
		t.Errorf("failed to find all: found %d", len(sdbs))
	}
}

func TestCleanAll(t *testing.T) {
	ctx := context.Background()
	client, truncate := connect(ctx, t, testConf.Config())
	defer truncate()
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

	if err := CleanAll(ctx, testConf.Config()); err != nil {
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
