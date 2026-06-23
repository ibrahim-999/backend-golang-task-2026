package query

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
)

type DailyReportResult struct {
	Date         string
	TotalOrders  int
	TotalRevenue int64
	Currency     string
	ByStatus     map[string]int
}

type DailyReport struct {
	reads     ports.RepositoryProvider
	pageSize  int
	maxOrders int
}

func NewDailyReport(reads ports.RepositoryProvider) *DailyReport {
	return &DailyReport{reads: reads, pageSize: 200, maxOrders: 2000}
}

func (q *DailyReport) Handle(ctx context.Context, date string) (DailyReportResult, error) {
	orders, err := q.collect(ctx)
	if err != nil {
		return DailyReportResult{}, err
	}

	var (
		totalOrders  int
		totalRevenue int64
		byStatus     map[string]int
		currency     = shared.DefaultCurrency
	)

	if len(orders) > 0 {
		currency = orders[0].Total().Currency()
	}

	group, _ := errgroup.WithContext(ctx)

	group.Go(func() error {
		totalOrders = len(orders)
		return nil
	})

	group.Go(func() error {
		var sum int64
		for _, o := range orders {
			sum += o.Total().Amount()
		}
		totalRevenue = sum
		return nil
	})

	group.Go(func() error {
		counts := make(map[string]int)
		for _, o := range orders {
			counts[string(o.Status())]++
		}
		byStatus = counts
		return nil
	})

	if err := group.Wait(); err != nil {
		return DailyReportResult{}, err
	}

	return DailyReportResult{
		Date:         date,
		TotalOrders:  totalOrders,
		TotalRevenue: totalRevenue,
		Currency:     currency,
		ByStatus:     byStatus,
	}, nil
}

func (q *DailyReport) collect(ctx context.Context) ([]*order.Order, error) {
	collected := make([]*order.Order, 0, q.pageSize)
	for page := 1; len(collected) < q.maxOrders; page++ {
		batch, total, err := q.reads.Orders().ListAll(ctx, ports.Page{Number: page, Size: q.pageSize})
		if err != nil {
			return nil, err
		}
		collected = append(collected, batch...)
		if len(batch) < q.pageSize || int64(len(collected)) >= total {
			break
		}
	}
	return collected, nil
}
