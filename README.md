# spool

[![CircleCI](https://circleci.com/gh/cloudspannerecosystem/spool.svg)](https://circleci.com/gh/cloudspannerecosystem/spool)

A CLI tool to manage [Cloud Spanner](https://cloud.google.com/spanner) databases for testing.

![spool](https://user-images.githubusercontent.com/2238852/68204102-a0764580-000a-11ea-879b-1acaf1c699c8.gif)

Please feel free to report issues and send pull requests, but note that this
application is not officially supported as part of the Cloud Spanner product.

## Motivation

When the development of spool started, the [Cloud Spanner
Emulator](https://cloud.google.com/spanner/docs/emulator) wasn't available yet.
When using Cloud Spanner instances for continuous integration tests, it is
inefficient to create a new test database on every run.
This tool lets you reuse test databases in CI tests.

## Installation

```shell
$ go get -u github.com/cloudspannerecosystem/spool/cmd/spool
```

## Setup

Spool requires a database for metadata to manage databases.
The following command sets up the database.

```shell
$ spool --project=${PROJECT} --instance=${INSTANCE} --database=${SPOOL_DATABASE} setup
```

## Usage

```shell
usage: spool [<flags>] <command> [<args> ...]

A CLI tool to manage Cloud Spanner databases for testing.

Flags:
      --help               Show context-sensitive help (also try --help-long and --help-man).
  -p, --project=PROJECT    Set GCP project ID. (use $SPANNER_PROJECT_ID or $GOOGLE_CLOUD_PROJECT as default value)
  -i, --instance=INSTANCE  Set Cloud Spanner instance name. (use $SPANNER_INSTANCE_ID as default value)
  -d, --database=DATABASE  Set Cloud Spanner database name. (use $SPOOL_SPANNER_DATABASE_ID as default value)
  -s, --schema=SCHEMA      Set schema file path.

Commands:
  help [<command>...]
    Show help.

  setup
    Setup the database for spool metadata.

  create --db-name-prefix=DB-NAME-PREFIX [<flags>]
    Add new databases to the pool.

  get
    Get a idle database from the pool.

  get-or-create --db-name-prefix=DB-NAME-PREFIX
    Get or create a idle database from the pool.

  list [<flags>]
    Print databases.

  put <database>
    Return the database to the pool.

  clean [<flags>]
    Drop all idle databases.
```

## Sample CircleCI configuration

```yaml
version: 2

jobs:
  build:
    docker:
      - image: golang:1.10-stretch
        environment:
          PROJECT: project
          INSTANCE: instance
          SPOOL_DATABASE: spool
          PATH_TO_SCHEMA_FILE: path/to/schema.sql
          DATABASE_PREFIX: spool
    working_directory: /go/src/github.com/user/repo
    steps:
      - checkout
      - run:
          name: set GitHub token
          command: |
            rm -f ~/.gitconfig
            echo "machine github.com login ${GITHUB_TOKEN}" > ~/.netrc
      - run:
          name: install spool
          command: go get -u github.com/cloudspannerecosystem/spool/cmd/spool
      - run:
          name: get database for testing
          command: |
            DATABASE=$(spool --project=${PROJECT} --instance=${INSTANCE} --database=${SPOOL_DATABASE} --schema=${PATH_TO_SCHEMA_FILE} get-or-create --db-name-prefix=${DATABASE_PREFIX})
            echo "export DATABASE=${DATABASE}" >> ${BASH_ENV}
      - run:
          name: run tests
          command: echo "run your tests with /projects/${PROJECT}/instances/${INSTANCE}/databases/${DATABASE}"
      - run:
          name: release database
          when: always
          command: spool --project=${PROJECT} --instance=${INSTANCE} --database=${SPOOL_DATABASE} --schema=${PATH_TO_SCHEMA_FILE} put ${DATABASE}

  cleanup-old-test-db:
    docker:
      - image: golang:1.10-stretch
        environment:
          PROJECT: project
          INSTANCE: instance
          SPOOL_DATABASE: spool
          PATH_TO_SCHEMA_FILE: path/to/schema.sql
    working_directory: /go/src/github.com/user/repo
    steps:
      - checkout
      - run:
          name: set GitHub token
          command: |
            rm -f ~/.gitconfig
            echo "machine github.com login ${GITHUB_TOKEN}" > ~/.netrc
      - run:
          name: install spool
          command: go get -u github.com/cloudspannerecosystem/spool/cmd/spool
      - run:
          name: cleanup databases
          command: spool --project=${PROJECT} --instance=${INSTANCE} --database=${SPOOL_DATABASE} --schema=${PATH_TO_SCHEMA_FILE} clean --all --force --ignore-used-within-days=7

workflows:
  version: 2
  build-workflow:
    jobs:
      - build:
          context: org-global
  cleanup-workflow:
    triggers:
      - schedule:
          cron: '0 9 * * *'
          filters:
            branches:
              only: master
    jobs:
      - cleanup-old-test-db:
          context: org-global
```
