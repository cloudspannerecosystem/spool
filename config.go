package spool

import (
	"fmt"

	"google.golang.org/api/option"
)

type Config struct {
	projectID  string
	instanceID string
	databaseID string
	opts       []option.ClientOption
}

func NewConfig(projectID, instanceID, databaseID string, opts ...option.ClientOption) *Config {
	return &Config{
		projectID:  projectID,
		instanceID: instanceID,
		databaseID: databaseID,
		opts:       opts,
	}
}

func (c *Config) ProjectID() string {
	return c.projectID
}

func (c *Config) InstanceID() string {
	return c.instanceID
}

func (c *Config) DatabaseID() string {
	return c.databaseID
}

func (c *Config) Instance() string {
	return fmt.Sprintf("projects/%s/instances/%s", c.projectID, c.instanceID)
}

func (c *Config) Database() string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s", c.projectID, c.instanceID, c.databaseID)
}

func (c *Config) ClientOptions() []option.ClientOption {
	return c.opts
}

func (c *Config) WithDatabaseID(databaseID string) *Config {
	return NewConfig(c.projectID, c.instanceID, databaseID, c.ClientOptions()...)
}
