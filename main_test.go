package spool

import (
	"context"
	"os"
	"testing"

	"github.com/cloudspannerecosystem/handy-spanner/fake"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
)

var (
	testConf config
	ddl1     []byte
	ddl2     []byte
)

type config struct {
	SpannerProjectID     string `envconfig:"SPANNER_PROJECT_ID" default:"fake"`
	SpannerInstanceID    string `envconfig:"SPANNER_INSTANCE_ID" default:"fake"`
	SpannerDatabaseID    string `envconfig:"SPANNER_DATABASE_ID" default:"fake"`
	DatabaseNamePrefix   string `envconfig:"SPOOL_SPANNER_DATABASE_NAME_PREFIX" default:"spool-test"`
	SpannerClientOptions []option.ClientOption
}

func (c config) Config() *Config {
	return NewConfig(c.SpannerProjectID, c.SpannerInstanceID, c.SpannerDatabaseID, c.SpannerClientOptions...)
}

func TestMain(m *testing.M) {
	if err := envconfig.Process("", &testConf); err != nil {
		panic(err)
	}

	srv, conn, err := fake.Run()
	if err != nil {
		panic(err)
	}
	defer srv.Stop()
	defer conn.Close()
	testConf.SpannerClientOptions = append(testConf.SpannerClientOptions, option.WithGRPCConn(conn))

	if err := Setup(context.Background(), testConf.Config()); err != nil {
		panic(errors.Wrap(err, "failed to setup spool metadata database"))
	}

	ddl1, err = readFile("testdata/schema1.sql")
	if err != nil {
		panic(err)
	}
	ddl2, err = readFile("testdata/schema2.sql")
	if err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
