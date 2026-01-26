package handler

import (
	"context"
)

func (h *Handler) RunAccrualWorker(ctx context.Context) error {
	h.log.Info("accrual worker started")
	h.worker.Run(ctx)
	return nil
}
