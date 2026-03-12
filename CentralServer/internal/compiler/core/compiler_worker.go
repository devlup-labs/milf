package core

import (
	"central_server/utils"
	"context"
	"fmt"
)

func (c *Compiler) Start(ctx context.Context) {
	utils.Info("[CompilerWorker] Compiler worker started...")
	for {
		select {
		case <-ctx.Done():
			utils.Info("[CompilerWorker] Compiler worker stopping...")
			return
		default:
			// 1. Dequeue job (blocking wait)
			job := c.queue.Dequeue()
			if job == nil {
				continue
			}

			utils.Info(fmt.Sprintf("[CompilerWorker] Dequeued job for FuncID: %s", job.FuncID))

			// 2. Compile
			utils.Info(fmt.Sprintf("[CompilerWorker] Starting compilation for %s", job.FuncID))
			_, compErr := c.Compile(job.FuncID)
			if compErr != nil {
				utils.Error(fmt.Sprintf("[CompilerWorker] Compilation failed for %s: %v", job.FuncID, compErr))
				// TODO: Update status to failed in DB?
				continue
			}
			utils.Info(fmt.Sprintf("[CompilerWorker] Compilation successful for %s", job.FuncID))

			// 3. Activate Service via Orchestrator
			utils.Info(fmt.Sprintf("[CompilerWorker] Activating service %s in Orchestrator", job.FuncID))
			_, err := c.orchestrator.ActivateService(ctx, job.FuncID)
			if err != nil {
				utils.Error(fmt.Sprintf("[CompilerWorker] Failed to activate service %s: %v", job.FuncID, err))
			} else {
				utils.Info(fmt.Sprintf("[CompilerWorker] Service %s activated successfully.", job.FuncID))
			}
		}
	}
}
