package spool

import (
	"time"

	"github.com/cloudspannerecosystem/spool/model"
)

// FilterNotUsedWithin returns a function which reports whether sdb is not used within d.
func FilterNotUsedWithin(d time.Duration) func(sdb *model.SpoolDatabase) bool {
	return func(sdb *model.SpoolDatabase) bool {
		return !sdb.UpdatedAt.Add(d).After(time.Now())
	}
}

// FilterState returns a function which reports whether sdb.State is state.
func FilterState(state State) func(sdb *model.SpoolDatabase) bool {
	return func(sdb *model.SpoolDatabase) bool {
		return sdb.State == state.Int64()
	}
}

func filter(sdbs []*model.SpoolDatabase, filters ...func(sdb *model.SpoolDatabase) bool) []*model.SpoolDatabase {
	res := make([]*model.SpoolDatabase, 0, len(sdbs))
	for _, sdb := range sdbs {
		var skip bool
		for _, filter := range filters {
			if !filter(sdb) {
				skip = true
				break
			}
		}
		if !skip {
			res = append(res, sdb)
		}
	}
	return res
}
