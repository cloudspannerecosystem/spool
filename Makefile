SPANNER_PROJECT_ID ?= spool-test-project
SPANNER_INSTANCE_ID ?= spool-test-instance
SPOOL_SPANNER_DATABASE_ID ?= spool-test-database

SPANNER_EMULATOR_HOST ?= localhost:9010
SPANNER_EMULATOR_HOST_REST ?= localhost:9020

BIN_DIR := .bin
YO_BIN := ${BIN_DIR}/yo
LINT_BIN := ${BIN_DIR}/golangci-lint
WRENCH_BIN := ${BIN_DIR}/wrench

${YO_BIN}:
	@mkdir -p ${BIN_DIR}
	@go build -o ${YO_BIN} go.mercari.io/yo

${LINT_BIN}:
	@mkdir -p ${BIN_DIR}
	@go build -o ${LINT_BIN} github.com/golangci/golangci-lint/cmd/golangci-lint

${WRENCH_BIN}:
	@mkdir -p ${BIN_DIR}
	@go build -o ${WRENCH_BIN} github.com/cloudspannerecosystem/wrench


.PHONY: install
install: ${WRENCH_BIN} ${YO_BIN} ${LINT_BIN}

.PHONY: clean
clean:
	rm -rf ${BIN_DIR}

.PHONY: gen
gen: gen_model

.PHONY: gen_model
gen_model: ${YO_BIN}
	rm -f ./model/*.yo.go
	${YO_BIN} $(SPANNER_PROJECT_ID) $(SPANNER_INSTANCE_ID) $(SPOOL_SPANNER_DATABASE_ID) --out ./model/

.PHONY: lint
lint: ${LINT_BIN}
	${LINT_BIN} run

.PHONY: test
test:
	go test -v -race -p=1 `go list ./... | grep -v tools`

.PHONY: setup-emulator
setup-emulator:
	curl -s "${SPANNER_EMULATOR_HOST_REST}/v1/projects/${SPANNER_PROJECT_ID}/instances" --data '{"instanceId": "'${SPANNER_INSTANCE_ID}'"}'

.PHONY: create_db
create_db: ${WRENCH_BIN}
	${WRENCH_BIN} create --project $(SPANNER_PROJECT_ID) --instance $(SPANNER_INSTANCE_ID) --database $(SPOOL_SPANNER_DATABASE_ID) --directory db/

.PHONY: drop_db
drop_db: ${WRENCH_BIN}
	${WRENCH_BIN} drop --project $(SPANNER_PROJECT_ID) --instance $(SPANNER_INSTANCE_ID) --database $(SPOOL_SPANNER_DATABASE_ID)

.PHONY: reset_db
reset_db:
	make drop_db
	make create_db

