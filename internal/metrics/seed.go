package metrics

import (
	"context"

	"github.com/go-tangra/go-tangra-hr/internal/data"
)

// Seed loads initial gauge values from the database.
// Called once at startup so Prometheus has accurate values from the start.
func (c *Collector) Seed(ctx context.Context, statsRepo *data.StatisticsRepo) {
	c.log.Info("Seeding Prometheus metrics from database...")

	absenceTypeCount, err := statsRepo.GetAbsenceTypeCount(ctx)
	if err != nil {
		c.log.Errorf("Failed to seed absence type stats: %v", err)
	} else {
		c.AbsenceTypesTotal.Set(float64(absenceTypeCount))
	}

	leaveRequestsByStatus, err := statsRepo.GetLeaveRequestCountByStatus(ctx)
	if err != nil {
		c.log.Errorf("Failed to seed leave request stats: %v", err)
	} else {
		for status, count := range leaveRequestsByStatus {
			c.LeaveRequestsByStatus.WithLabelValues(status).Set(float64(count))
		}
	}

	allowanceCount, err := statsRepo.GetLeaveAllowanceCount(ctx)
	if err != nil {
		c.log.Errorf("Failed to seed leave allowance stats: %v", err)
	} else {
		c.LeaveAllowancesTotal.Set(float64(allowanceCount))
	}

	c.log.Info("Prometheus metrics seeded successfully")
}
