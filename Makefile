SPANNER_PROJECT_ID  ?=
SPANNER_INSTANCE_ID ?=
SPOOL_SPANNER_DATABASE_ID ?=

BIN_DIR := .bin
YO_BIN := ${BIN_DIR}/yo
STATIK_BIN := ${BIN_DIR}/statik
LINT_BIN := ${BIN_DIR}/golangci-lint
WRENCH_BIN := ${BIN_DIR}/wrench

${YO_BIN}:
	@mkdir -p ${BIN_DIR}
	@go build -o ${YO_BIN} go.mercari.io/yo

${STATIK_BIN}:
	@mkdir -p ${BIN_DIR}
	@go build -o ${STATIK_BIN} github.com/rakyll/statik

${LINT_BIN}:
	@mkdir -p ${BIN_DIR}
	@go build -o ${LINT_BIN} github.com/golangci/golangci-lint/cmd/golangci-lint

${WRENCH_BIN}:
	@mkdir -p ${BIN_DIR}
	@go build -o ${WRENCH_BIN} github.com/cloudspannerecosystem/wrench


.PHONY: install
install: ${WRENCH_BIN} ${YO_BIN} ${STATIK_BIN} ${LINT_BIN}

.PHONY: clean
clean:
	rm -rf ${BIN_DIR}

.PHONY: gen
gen: gen_model gen_assets

.PHONY: gen_model
gen_model: ${YO_BIN}
	rm -f ./model/*.yo.go
	${YO_BIN} $(SPANNER_PROJECT_ID) $(SPANNER_INSTANCE_ID) $(SPOOL_SPANNER_DATABASE_ID) --out ./model/

.PHONY: gen_assets
gen_assets: ${STATIK_BIN}
	${STATIK_BIN} -src ./db

.PHONY: lint
lint: ${LINT_BIN}
	${LINT_BIN} run

.PHONY: test
test:
	go test -v -race -p=1 `go list ./... | grep -v tools`

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

