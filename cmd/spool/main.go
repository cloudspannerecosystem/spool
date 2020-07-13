package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"text/tabwriter"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/cloudspannerecosystem/spool"
	"github.com/cloudspannerecosystem/spool/model"
)

const (
	envProjectID            = "SPANNER_PROJECT_ID"
	envGoogleCloudProjectID = "GOOGLE_CLOUD_PROJECT"
	envInstanceID           = "SPANNER_INSTANCE_ID"
	envDatabaseID           = "SPOOL_SPANNER_DATABASE_ID"
)

var (
	app        = kingpin.New("spool", "A CLI tool to manage Cloud Spanner databases for testing.")
	projectID  = app.Flag("project", "Set GCP project ID. (use $SPANNER_PROJECT_ID or $GOOGLE_CLOUD_PROJECT as default value)").Short('p').String()
	instanceID = app.Flag("instance", "Set Cloud Spanner instance name. (use $SPANNER_INSTANCE_ID as default value)").Short('i').String()
	databaseID = app.Flag("database", "Set Cloud Spanner database name. (use $SPOOL_SPANNER_DATABASE_ID as default value)").Short('d').String()
	schemaFile = app.Flag("schema", "Set schema file path.").Short('s').File()

	setup = app.Command("setup", "Setup the database for spool metadata.")

	create                   = app.Command("create", "Add new databases to the pool.")
	createDatabaseNamePrefix = create.Flag("db-name-prefix", "Set new database name prefix.").Required().String()
	createDatabaseNum        = create.Flag("num", "Set the number of new databases.").Default("1").Int()

	get = app.Command("get", "Get a idle database from the pool.")

	getOrCreate                   = app.Command("get-or-create", "Get or create a idle database from the pool.")
	getOrCreateDatabaseNamePrefix = getOrCreate.Flag("db-name-prefix", "Set new database name prefix.").Required().String()

	list    = app.Command("list", "Print databases.")
	listAll = list.Flag("all", "Print databases. (without checksum filtering)").Default("false").Bool()

	put             = app.Command("put", "Return the database to the pool.")
	putDatabaseName = put.Arg("database", "database name").Required().String()

	clean                     = app.Command("clean", "Drop all idle databases.")
	cleanAll                  = clean.Flag("all", "Drop all idle databases. (without checksum filtering)").Default("false").Bool()
	cleanIgnoreUsedWithinDays = clean.Flag("ignore-used-within-days", "Ignore databases which used within n days.").Int64()
	cleanForce                = clean.Flag("force", "Drop all databases. (include busy databases)").Default("false").Bool()
)

func main() {
	ctx := context.Background()
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))
	if err := loadEnvVarsIfNeeded(); err != nil {
		kingpin.Fatalf("%s, try --help", err)
	}

	config := spool.NewConfig(*projectID, *instanceID, *databaseID)

	switch cmd {
	case setup.FullCommand():
		kingpin.FatalIfError(spool.Setup(ctx, config), "failed to setup")
	case create.FullCommand():
		pool := newPool(ctx, config)
		for i := 0; i < *createDatabaseNum; i++ {
			_, err := pool.Create(ctx, *createDatabaseNamePrefix)
			kingpin.FatalIfError(err, "failed to create database")
		}
	case get.FullCommand():
		pool := newPool(ctx, config)
		sdb, err := pool.Get(ctx)
		kingpin.FatalIfError(err, "failed to get database")
		fmt.Print(sdb.DatabaseName)
	case getOrCreate.FullCommand():
		pool := newPool(ctx, config)
		sdb, err := pool.GetOrCreate(ctx, *getOrCreateDatabaseNamePrefix)
		kingpin.FatalIfError(err, "failed to get or create database")
		fmt.Print(sdb.DatabaseName)
	case list.FullCommand():
		var sdbs []*model.SpoolDatabase
		var err error
		if *listAll {
			sdbs, err = spool.ListAll(ctx, config)
		} else {
			pool := newPool(ctx, config)
			sdbs, err = pool.List(ctx)
		}
		kingpin.FatalIfError(err, "failed to get databases")
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
		for _, sdb := range sdbs {
			_, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", sdb.DatabaseName, sdb.Checksum, spool.State(sdb.State), sdb.CreatedAt.In(time.Local), sdb.UpdatedAt.In(time.Local))
			kingpin.FatalIfError(err, "failed to print databases")
		}
		if err := w.Flush(); err != nil {
			kingpin.FatalIfError(err, "failed to print databases")
		}
	case put.FullCommand():
		pool := newPool(ctx, config)
		err := pool.Put(ctx, *putDatabaseName)
		kingpin.FatalIfError(err, "failed to put database")
	case clean.FullCommand():
		filters := []func(*model.SpoolDatabase) bool{}
		if cleanIgnoreUsedWithinDays != nil {
			filters = append(filters, spool.FilterNotUsedWithin(time.Duration(*cleanIgnoreUsedWithinDays)*24*time.Hour))
		}
		if !*cleanForce {
			filters = append(filters, spool.FilterState(spool.StateIdle))
		}
		var err error
		if *cleanAll {
			err = spool.CleanAll(ctx, config, filters...)
		} else {
			pool := newPool(ctx, config)
			err = pool.Clean(ctx, filters...)
		}
		kingpin.FatalIfError(err, "failed to clean database")
	}
}

func loadEnvVarsIfNeeded() error {
	if projectID == nil || *projectID == "" {
		if v, ok := os.LookupEnv(envProjectID); ok {
			projectID = &v
		} else if v, ok := os.LookupEnv(envGoogleCloudProjectID); ok {
			projectID = &v
		} else {
			return errors.New("required flag --project not provided")
		}
	}
	if instanceID == nil || *instanceID == "" {
		if v, ok := os.LookupEnv(envInstanceID); ok {
			instanceID = &v
		} else {
			return errors.New("required flag --instance not provided")
		}
	}
	if databaseID == nil || *databaseID == "" {
		if v, ok := os.LookupEnv(envDatabaseID); ok {
			databaseID = &v
		} else {
			return errors.New("required flag --database not provided")
		}
	}
	return nil
}

func newPool(ctx context.Context, config *spool.Config) *spool.Pool {
	if *schemaFile == nil {
		kingpin.Fatalf("required flag --schema not provided, try --help")
	}
	ddl, err := ioutil.ReadAll(*schemaFile)
	kingpin.FatalIfError(err, "failed to read schema file")
	pool, err := spool.NewPool(ctx, config, ddl)
	kingpin.FatalIfError(err, "")
	return pool
}
