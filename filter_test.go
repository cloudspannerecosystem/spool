package spool

import (
	"testing"
	"time"

	"github.com/cloudspannerecosystem/spool/model"
)

func TestFilterNotUsedWithin(t *testing.T) {
	now := time.Now()
	hour := time.Hour
	sdbs := []*model.SpoolDatabase{
		{
			UpdatedAt: now.Add(-hour),
		},
		{
			UpdatedAt: now.Add(-hour).Add(time.Second),
		},
	}
	filtered := filter(sdbs, FilterNotUsedWithin(hour))
	if len(filtered) != 1 {
		t.Errorf("expected 1 but got %d", len(filtered))
	} else if !filtered[0].UpdatedAt.Equal(sdbs[0].UpdatedAt) {
		t.Errorf("expected %s but got %s", sdbs[0].UpdatedAt, filtered[0].UpdatedAt)
	}
}

func TestFilterState(t *testing.T) {
	sdbs := []*model.SpoolDatabase{
		{
			State: StateIdle.Int64(),
		},
		{
			State: StateBusy.Int64(),
		},
	}
	state := StateIdle
	filtered := filter(sdbs, FilterState(state))
	if len(filtered) != 1 {
		t.Errorf("expected 1 but got %d", len(filtered))
	} else if filtered[0].State != state.Int64() {
		t.Errorf("expected %s but got %s", state, State(filtered[0].State))
	}
}
