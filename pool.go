package spool

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	admin "cloud.google.com/go/spanner/admin/database/apiv1"
	"github.com/gcpug/spool/model"
	databasepb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
)

// State represents a state of the database.
type State int64

const (
	// StateIdle represents a idle state.
	StateIdle State = iota
	// StateBusy represents a busy state.
	StateBusy
)

// Int64 returns s as int64.
func (s State) Int64() int64 {
	return int64(s)
}

// String returns a string representing the state.
func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateBusy:
		return "busy"
	}
	return "unknown"
}

// Pool represents a spanner database pool.
type Pool struct {
	client        *spanner.Client
	adminClient   *admin.DatabaseAdminClient
	conf          *Config
	ddlStatements []string
	checksum      string
}

// NewPool creates a new Pool.
func NewPool(ctx context.Context, conf *Config, ddl []byte) (*Pool, error) {
	client, err := spanner.NewClient(ctx, conf.Database(), conf.ClientOptions()...)
	if err != nil {
		return nil, err
	}
	adminClient, err := admin.NewDatabaseAdminClient(ctx, conf.ClientOptions()...)
	if err != nil {
		return nil, err
	}
	pool := &Pool{
		client:        client,
		adminClient:   adminClient,
		conf:          conf,
		ddlStatements: ddlToStatements(ddl),
		checksum:      checksum(ddl),
	}
	return pool, nil
}

func ddlToStatements(ddl []byte) []string {
	ddls := bytes.Split(ddl, []byte(";"))
	ddlStatements := make([]string, 0, len(ddls))
	for _, s := range ddls {
		if stmt := strings.TrimSpace(string(s)); stmt != "" {
			ddlStatements = append(ddlStatements, stmt)
		}
	}
	return ddlStatements
}

func checksum(ddl []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(ddl))
}

// Create creates a new database and adds to the pool.
func (p *Pool) Create(ctx context.Context, dbNamePrefix string) (*model.SpoolDatabase, error) {
	dbName := fmt.Sprintf("%s-%d", dbNamePrefix, time.Now().Unix())
	sdb := &model.SpoolDatabase{
		DatabaseName: dbName,
		Checksum:     p.checksum,
		State:        StateIdle.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	return p.create(ctx, sdb)
}

func (p *Pool) create(ctx context.Context, sdb *model.SpoolDatabase) (*model.SpoolDatabase, error) {
	op, err := p.adminClient.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          p.conf.Instance(),
		CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", sdb.DatabaseName),
		ExtraStatements: p.ddlStatements,
	})
	if err != nil {
		return nil, err
	}
	if _, err := op.Wait(ctx); err != nil {
		return nil, err
	}
	ts, err := p.client.Apply(ctx, []*spanner.Mutation{sdb.Insert(ctx)})
	if err != nil {
		_ = dropDatabase(ctx, p.conf.WithDatabaseID(sdb.DatabaseName))
		return nil, err
	}
	sdb.CreatedAt = ts
	sdb.UpdatedAt = ts
	return sdb, nil
}

// Get gets a idle database from the pool.
func (p *Pool) Get(ctx context.Context) (*model.SpoolDatabase, error) {
	var sdb *model.SpoolDatabase
	if _, err := p.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var err error
		sdb, err = model.FindSpoolDatabaseByChecksumState(ctx, txn, p.checksum, StateIdle.Int64())
		if err != nil {
			return err
		}
		sdb.ChangeState(StateBusy.Int64())
		if err := txn.BufferWrite([]*spanner.Mutation{sdb.Update(ctx)}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return sdb, nil
}

// GetOrCreate gets a idle database or creates a new database.
func (p *Pool) GetOrCreate(ctx context.Context, dbNamePrefix string) (*model.SpoolDatabase, error) {
	sdb, err := p.Get(ctx)
	if err == nil {
		return sdb, nil
	}
	if !isErrNotFound(err) {
		return nil, err
	}
	dbName := fmt.Sprintf("%s-%d", dbNamePrefix, time.Now().Unix())
	sdb = &model.SpoolDatabase{
		DatabaseName: dbName,
		Checksum:     p.checksum,
		State:        StateBusy.Int64(),
		CreatedAt:    spanner.CommitTimestamp,
		UpdatedAt:    spanner.CommitTimestamp,
	}
	return p.create(ctx, sdb)
}

// List gets all databases from the pool.
func (p *Pool) List(ctx context.Context) ([]*model.SpoolDatabase, error) {
	return model.FindSpoolDatabasesByChecksum(ctx, p.client.ReadOnlyTransaction(), p.checksum)
}

// Put adds a database to the pool.
func (p *Pool) Put(ctx context.Context, dbName string) error {
	if _, err := p.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		sdb, err := model.FindSpoolDatabase(ctx, txn, dbName)
		if err != nil {
			return err
		}
		sdb.ChangeState(StateIdle.Int64())
		if err := txn.BufferWrite([]*spanner.Mutation{sdb.Update(ctx)}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// Clean removes all idle databases.
func (p *Pool) Clean(ctx context.Context, filters ...func(sdb *model.SpoolDatabase) bool) error {
	return clean(ctx, p.client, p.conf, func(ctx context.Context, txn *spanner.ReadWriteTransaction) ([]*model.SpoolDatabase, error) {
		sdbs, err := model.FindSpoolDatabasesByChecksumState(ctx, txn, p.checksum, StateIdle.Int64())
		if err != nil {
			return nil, err
		}
		return filter(sdbs, filters...), nil
	})
}
