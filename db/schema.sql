CREATE TABLE SpoolDatabases (
  DatabaseName STRING(MAX) NOT NULL,
  Checksum STRING(MAX) NOT NULL,
  State INT64 NOT NULL,
  CreatedAt TIMESTAMP NOT NULL OPTIONS (
    allow_commit_timestamp = true
  ),
  UpdatedAt TIMESTAMP NOT NULL OPTIONS (
    allow_commit_timestamp = true
  ),
) PRIMARY KEY(DatabaseName);

CREATE INDEX SpoolDatabasesByChecksumAndState ON SpoolDatabases(Checksum, State);
