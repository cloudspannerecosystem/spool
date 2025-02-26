package spool

import (
	"context"
	"io"
	"os"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/cloudspannerecosystem/spool/model"
)

func newPool(ctx context.Context, t *testing.T, ddl []byte) *Pool {
	t.Helper()
	pool, err := NewPool(ctx, testConf.Config(), ddl)
	if err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestPool_Create(t *testing.T) {
	ctx := context.Background()
	client, truncate := connect(ctx, t, testConf.Config())
	defer truncate()

	pool := newPool(ctx, t, ddl1)
	sdb, err := pool.Create(ctx, testConf.DatabaseNamePrefix)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := model.FindSpoolDatabase(ctx, client.Single(), sdb.DatabaseName); err != nil {
		t.Error(err)
	}
}

func TestPool_Get(t *testing.T) {
	ctx := context.Background()
	client, truncate := connect(ctx, t, testConf.Config())
	defer truncate()

	pool := newPool(ctx, t, ddl1)
	sdb := &model.SpoolDatabase{
		DatabaseName: "zoncoen-spool-test",
		Checksum:     checksum(ddl1),
		State:        StateIdle.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	if _, err := client.Apply(ctx, []*spanner.Mutation{sdb.Insert(ctx)}); err != nil {
		t.Fatalf("failed to setup fixture: %s", err)
	}

	t.Run("not found (no database for the schema)", func(t *testing.T) {
		pool2 := newPool(ctx, t, ddl2)
		if _, err := pool2.Get(ctx); err != nil {
			if !isErrNotFound(err) {
				t.Fatal(err)
			}
		} else {
			t.Fatal("should not get another schema database")
		}
	})
	t.Run("found", func(t *testing.T) {
		if _, err := pool.Get(ctx); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("not found (already used)", func(t *testing.T) {
		if _, err := pool.Get(ctx); err != nil {
			if !isErrNotFound(err) {
				t.Fatal(err)
			}
		} else {
			t.Fatal("should not get busy database")
		}
	})
}

func TestPool_GetOrCreate(t *testing.T) {
	ctx := context.Background()
	client, truncate := connect(ctx, t, testConf.Config())
	defer truncate()

	pool := newPool(ctx, t, ddl1)
	sdb := &model.SpoolDatabase{
		DatabaseName: "zoncoen-spool-test",
		Checksum:     checksum(ddl1),
		State:        StateIdle.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	if _, err := client.Apply(ctx, []*spanner.Mutation{sdb.Insert(ctx)}); err != nil {
		t.Fatalf("failed to setup fixture: %s", err)
	}

	t.Run("get", func(t *testing.T) {
		got, err := pool.GetOrCreate(ctx, testConf.DatabaseNamePrefix)
		if err != nil {
			t.Fatal(err)
		}
		if got.DatabaseName != sdb.DatabaseName {
			t.Errorf("expected %s but got %s", sdb.DatabaseName, got.DatabaseName)
		}
	})
	t.Run("create", func(t *testing.T) {
		got, err := pool.GetOrCreate(ctx, testConf.DatabaseNamePrefix)
		if err != nil {
			t.Fatal(err)
		}
		if got.DatabaseName == sdb.DatabaseName {
			t.Error("should not get busy database")
		}
	})
}

func TestPool_List(t *testing.T) {
	ctx := context.Background()
	client, truncate := connect(ctx, t, testConf.Config())
	defer truncate()

	pool := newPool(ctx, t, ddl1)
	sdb1 := &model.SpoolDatabase{
		DatabaseName: "zoncoen-spool-test-1",
		Checksum:     checksum(ddl1),
		State:        StateIdle.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	sdb2 := &model.SpoolDatabase{
		DatabaseName: "zoncoen-spool-test-2",
		Checksum:     checksum(ddl2),
		State:        StateIdle.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	if _, err := client.Apply(ctx, []*spanner.Mutation{sdb1.Insert(ctx), sdb2.Insert(ctx)}); err != nil {
		t.Fatalf("failed to setup fixture: %s", err)
	}

	sdbs, err := pool.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sdbs) != 1 {
		t.Fatalf("expected 1 but gut %d", len(sdbs))
	}
	if sdbs[0].DatabaseName != sdb1.DatabaseName {
		t.Errorf("expected %s but got %s", sdb1.DatabaseName, sdbs[0].DatabaseName)
	}
}

func TestPool_Put(t *testing.T) {
	ctx := context.Background()
	client, truncate := connect(ctx, t, testConf.Config())
	defer truncate()

	pool := newPool(ctx, t, ddl1)
	sdb := &model.SpoolDatabase{
		DatabaseName: "zoncoen-spool-test",
		Checksum:     checksum(ddl1),
		State:        StateBusy.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	if _, err := client.Apply(ctx, []*spanner.Mutation{sdb.Insert(ctx)}); err != nil {
		t.Fatalf("failed to setup fixture: %s", err)
	}

	if err := pool.Put(ctx, sdb.DatabaseName); err != nil {
		t.Fatal(err)
	}
	got, err := model.FindSpoolDatabase(ctx, client.Single(), sdb.DatabaseName)
	if err != nil {
		t.Error(err)
	}
	if state := State(got.State); state != StateIdle {
		t.Errorf("expected %s but got %s", StateIdle, state)
	}
}

func TestPool_Clean(t *testing.T) {
	ctx := context.Background()
	client, truncate := connect(ctx, t, testConf.Config())
	defer truncate()

	pool := newPool(ctx, t, ddl1)
	sdb1 := &model.SpoolDatabase{
		DatabaseName: "zoncoen-spool-test-1",
		Checksum:     checksum(ddl1),
		State:        StateIdle.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	sdb2 := &model.SpoolDatabase{
		DatabaseName: "zoncoen-spool-test-2",
		Checksum:     checksum(ddl2),
		State:        StateIdle.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	if _, err := client.Apply(ctx, []*spanner.Mutation{sdb1.Insert(ctx), sdb2.Insert(ctx)}); err != nil {
		t.Fatalf("failed to setup fixture: %s", err)
	}

	if err := pool.Clean(ctx); err != nil {
		t.Fatal(err)
	}
	t.Run("should be deleted", func(t *testing.T) {
		if _, err := model.FindSpoolDatabase(ctx, client.Single(), sdb1.DatabaseName); err != nil {
			if !isErrNotFound(err) {
				t.Fatal(err)
			}
		} else {
			t.Fatal("should be deleted")
		}
	})
	t.Run("another schema database should not be deleted", func(t *testing.T) {
		if _, err := model.FindSpoolDatabase(ctx, client.Single(), sdb2.DatabaseName); err != nil {
			t.Fatal(err)
		}
	})
}

func readFile(filepath string) ([]byte, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}
