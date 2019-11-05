package model

import (
	"context"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

func (sdb *SpoolDatabase) ChangeState(state int64) {
	sdb.State = state
	sdb.UpdatedAt = spanner.CommitTimestamp
}

// FindAllSpoolDatabases finds all SpoolDatabases.
func FindAllSpoolDatabases(ctx context.Context, db YORODB) ([]*SpoolDatabase, error) {
	const sqlstr = `SELECT ` +
		`* ` +
		`FROM SpoolDatabases`

	stmt := spanner.NewStatement(sqlstr)
	customPtrs := make(map[string]interface{}, 0)

	// run query
	YOLog(ctx, sqlstr)
	iter := db.Query(ctx, stmt)
	defer iter.Stop()

	// load results
	res := []*SpoolDatabase{}
	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, newError("FindSpoolDatabasesByChecksumState", "SpoolDatabases", err)
		}

		var sd SpoolDatabase
		ptrs, err := sd.columnsToPtrs(SpoolDatabaseColumns(), customPtrs)
		if err != nil {
			return nil, newError("FindSpoolDatabasesByChecksumState", "SpoolDatabases", err)
		}

		if err := row.Columns(ptrs...); err != nil {
			return nil, newErrorWithCode(codes.Internal, "FindSpoolDatabasesByChecksumState", "SpoolDatabases", err)
		}

		res = append(res, &sd)
	}

	return res, nil
}

// FindSpoolDatabasesByChecksum finds a SpoolDatabase by Checksum.
func FindSpoolDatabasesByChecksum(ctx context.Context, db YORODB, checksum string) ([]*SpoolDatabase, error) {
	const sqlstr = `SELECT ` +
		`*` +
		`FROM SpoolDatabases@{FORCE_INDEX=SpoolDatabasesByChecksumAndState} ` +
		`WHERE Checksum = @param0`

	stmt := spanner.NewStatement(sqlstr)
	stmt.Params["param0"] = checksum
	customPtrs := make(map[string]interface{}, 0)

	// run query
	YOLog(ctx, sqlstr, checksum)
	iter := db.Query(ctx, stmt)
	defer iter.Stop()

	// load results
	res := []*SpoolDatabase{}
	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, newError("FindSpoolDatabasesByChecksumState", "SpoolDatabases", err)
		}

		var sd SpoolDatabase
		ptrs, err := sd.columnsToPtrs(SpoolDatabaseColumns(), customPtrs)
		if err != nil {
			return nil, newError("FindSpoolDatabasesByChecksumState", "SpoolDatabases", err)
		}

		if err := row.Columns(ptrs...); err != nil {
			return nil, newErrorWithCode(codes.Internal, "FindSpoolDatabasesByChecksumState", "SpoolDatabases", err)
		}

		res = append(res, &sd)
	}

	return res, nil
}

// FindSpoolDatabaseByChecksumState finds a SpoolDatabase by Checksum and State.
func FindSpoolDatabaseByChecksumState(ctx context.Context, db YORODB, checksum string, state int64) (*SpoolDatabase, error) {
	const sqlstr = `SELECT ` +
		`*` +
		`FROM SpoolDatabases@{FORCE_INDEX=SpoolDatabasesByChecksumAndState} ` +
		`WHERE Checksum = @param0 AND State = @param1 Limit 1`

	stmt := spanner.NewStatement(sqlstr)
	stmt.Params["param0"] = checksum
	stmt.Params["param1"] = state
	customPtrs := make(map[string]interface{}, 0)

	// run query
	YOLog(ctx, sqlstr, checksum, state)
	var sd SpoolDatabase
	ptrs, err := sd.columnsToPtrs(SpoolDatabaseColumns(), customPtrs)
	if err != nil {
		return nil, newError("FindSpoolDatabasesByChecksumState", "SpoolDatabases", err)
	}

	iter := db.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, newErrorWithCode(codes.NotFound, "FindSpoolDatabasesByChecksumState", "SpoolDatabases", err)
		}
		return nil, newError("FindSpoolDatabasesByChecksumState", "SpoolDatabases", err)
	}

	if err := row.Columns(ptrs...); err != nil {
		return nil, newErrorWithCode(codes.Internal, "FindSpoolDatabasesByChecksumState", "SpoolDatabases", err)
	}

	return &sd, nil
}
