package spool

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	admin "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/cloudspannerecosystem/spool/internal/db"
	"github.com/cloudspannerecosystem/spool/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	defer client.Close()
	adminClient, err := admin.NewDatabaseAdminClient(ctx, conf.ClientOptions()...)
	if err != nil {
		return err
	}
	defer func() { _ = adminClient.Close() }()
	return clean(ctx, client, adminClient, conf, func(ctx context.Context, txn *spanner.ReadOnlyTransaction) ([]*model.SpoolDatabase, error) {
		sdbs, err := model.FindAllSpoolDatabases(ctx, txn)
		if err != nil {
			return nil, err
		}
		return filter(sdbs, filters...), nil
	})
}

func clean(ctx context.Context, client *spanner.Client, adminClient *admin.DatabaseAdminClient, conf *Config, find func(ctx context.Context, txn *spanner.ReadOnlyTransaction) ([]*model.SpoolDatabase, error)) error {
	sdbs, err := find(ctx, client.Single())
	if err != nil {
		return err
	}
	for _, sdb := range sdbs {
		if _, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			latest, err := model.FindSpoolDatabase(ctx, txn, sdb.DatabaseName)
			if err != nil {
				if isErrNotFound(err) {
					// The row was already removed since it was listed.
					return nil
				}
				return err
			}
			if latest.State != sdb.State || !latest.UpdatedAt.Equal(sdb.UpdatedAt) {
				// The database was used since it was listed, so it may no longer be a clean target.
				return nil
			}
			if err := dropDatabase(ctx, adminClient, conf.WithDatabaseID(sdb.DatabaseName)); err != nil {
				if status.Code(err) != codes.NotFound {
					return err
				}
				fmt.Printf("%s was not deleted because it no longer exists.\n", sdb.DatabaseName)
			}
			return txn.BufferWrite([]*spanner.Mutation{sdb.Delete(ctx)})
		}); err != nil {
			return err
		}
	}
	return nil
}

func dropDatabase(ctx context.Context, adminClient *admin.DatabaseAdminClient, conf *Config) error {
	return adminClient.DropDatabase(ctx, &databasepb.DropDatabaseRequest{
		Database: conf.Database(),
	})
}
