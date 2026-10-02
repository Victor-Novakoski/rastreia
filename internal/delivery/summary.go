package delivery

import (
	"context"
	"time"

	"github.com/Victor-Novakoski/rastreia/internal/store"
)

// summaryWindow is how far back the dashboard counts deliveries.
const summaryWindow = 30 * 24 * time.Hour

// Summary is the carrier's dashboard: deliveries per status created since
// Since, and the pending ones still without a driver, whenever created.
type Summary struct {
	Since      time.Time        `json:"since"`
	ByStatus   map[string]int64 `json:"by_status"`
	Unassigned int64            `json:"unassigned"`
}

func (s *Service) Summary(ctx context.Context, carrierID int64) (Summary, error) {
	since := s.now().Add(-summaryWindow)
	rows, err := s.store.CountDeliveriesByStatus(ctx, store.CountDeliveriesByStatusParams{
		CarrierID: carrierID, Since: since,
	})
	if err != nil {
		return Summary{}, err
	}
	unassigned, err := s.store.CountUnassignedDeliveries(ctx, carrierID)
	if err != nil {
		return Summary{}, err
	}
	// Every status is present, so the front does not have to fill in zeros.
	out := Summary{Since: since, ByStatus: make(map[string]int64, len(statuses)), Unassigned: unassigned}
	for _, st := range statuses {
		out.ByStatus[st] = 0
	}
	for _, r := range rows {
		out.ByStatus[r.Status] = r.Total
	}
	return out, nil
}
