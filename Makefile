export SPANNER_PROJECT_ID ?= spool-test-project
export SPANNER_INSTANCE_ID ?= spool-test-instance
export SPOOL_SPANNER_DATABASE_ID ?= spool-test-database

export SPANNER_EMULATOR_HOST ?= localhost:9010
export SPANNER_EMULATOR_HOST_REST ?= localhost:9020

YO_BIN := go tool yo
LINT_BIN := go tool golangci-lint
WRENCH_BIN := go tool wrench

BIN_DIR := .bin

.PHONY: clean
clean:
	rm -rf ${BIN_DIR}

.PHONY: gen
gen: gen_model

.PHONY: gen_model
gen_model:
	rm -f ./model/*.yo.go
	${YO_BIN} $(SPANNER_PROJECT_ID) $(SPANNER_INSTANCE_ID) $(SPOOL_SPANNER_DATABASE_ID) --out ./model/

.PHONY: lint
lint:
	${LINT_BIN} run

.PHONY: test
test:
	go test -v -race -p=1 `go list ./...`

.PHONY: setup-emulator
setup-emulator:
	curl -s "${SPANNER_EMULATOR_HOST_REST}/v1/projects/${SPANNER_PROJECT_ID}/instances" --data '{"instanceId": "'${SPANNER_INSTANCE_ID}'"}'

.PHONY: create_db
create_db:
	${WRENCH_BIN} create --project $(SPANNER_PROJECT_ID) --instance $(SPANNER_INSTANCE_ID) --database $(SPOOL_SPANNER_DATABASE_ID) --directory db/

.PHONY: drop_db
drop_db:
	${WRENCH_BIN} drop --project $(SPANNER_PROJECT_ID) --instance $(SPANNER_INSTANCE_ID) --database $(SPOOL_SPANNER_DATABASE_ID)

.PHONY: reset_db
reset_db:
	make drop_db
	make create_db
