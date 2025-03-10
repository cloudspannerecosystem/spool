package spool

import (
	"context"
	"fmt"
	"github.com/cloudspannerecosystem/spool/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cloud.google.com/go/spanner"
	admin "cloud.google.com/go/spanner/admin/database/apiv1"
	"github.com/cloudspannerecosystem/spool/model"
	databasepb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
)

// Setup creates a new spool metadata database.
func Setup(ctx context.Context, conf *Config) error {
	adminClient, err := admin.NewDatabaseAdminClient(ctx, conf.ClientOptions()...)
	if err != nil {
		return err
	}

	_, err = adminClient.GetDatabase(ctx, &databasepb.GetDatabaseRequest{
		Name: conf.Database(),
	})
	if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
		// Database does not exist. Create a new one.
		op, err := adminClient.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
			Parent:          conf.Instance(),
			CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", conf.DatabaseID()),
			ExtraStatements: ddlToStatements(db.SpoolSchema),
		})
		if err != nil {
			return err
		}
		if _, err := op.Wait(ctx); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Database already exists. Try to update schema.
		// Considerations when the database is created using terraform, etc.
		op, err := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database:   conf.Database(),
			Statements: ddlToStatements(db.SpoolSchema),
		})
		if err != nil {
			return err
		}
		if err := op.Wait(ctx); err != nil {
			return err
		}
	}

	return nil
}

// ListAll gets all databases from the pool.
func ListAll(ctx context.Context, conf *Config) ([]*model.SpoolDatabase, error) {
	client, err := spanner.NewClient(ctx, conf.Database(), conf.ClientOptions()...)
	if err != nil {
		return nil, err
	}
	return model.FindAllSpoolDatabases(ctx, client.ReadOnlyTransaction())
}

// CleanAll removes all idle databases.
func CleanAll(ctx context.Context, conf *Config, filters ...func(sdb *model.SpoolDatabase) bool) error {
	client, err := spanner.NewClient(ctx, conf.Database(), conf.ClientOptions()...)
	if err != nil {
		return err
	}
	return clean(ctx, client, conf, func(ctx context.Context, txn *spanner.ReadWriteTransaction) ([]*model.SpoolDatabase, error) {
		sdbs, err := model.FindAllSpoolDatabases(ctx, txn)
		if err != nil {
			return nil, err
		}
		return filter(sdbs, filters...), nil
	})
}

func clean(ctx context.Context, client *spanner.Client, conf *Config, find func(ctx context.Context, txn *spanner.ReadWriteTransaction) ([]*model.SpoolDatabase, error)) error {
	var dropErr error
	if _, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		sdbs, err := find(ctx, txn)
		if err != nil {
			return err
		}
		ms := []*spanner.Mutation{}
		for _, sdb := range sdbs {
			dropErr = dropDatabase(ctx, conf.WithDatabaseID(sdb.DatabaseName))
			if dropErr != nil {
				break
			}
			ms = append(ms, sdb.Delete(ctx))
		}
		if len(ms) > 0 {
			if err := txn.BufferWrite(ms); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if dropErr != nil {
		return dropErr
	}
	return nil
}

func dropDatabase(ctx context.Context, conf *Config) error {
	adminClient, err := admin.NewDatabaseAdminClient(ctx, conf.ClientOptions()...)
	if err != nil {
		return err
	}
	return adminClient.DropDatabase(ctx, &databasepb.DropDatabaseRequest{
		Database: conf.Database(),
	})
}
