package core

import (
	interfaces "central_server/internal/orchestrator/interface"
	"central_server/utils"
	"context"
	"errors"

	"github.com/google/uuid"
)

type Orchestrator struct {
	OrchestratorID string
	Activated      map[string]bool
	Funcs          map[string]map[string]string
	trigJob        map[string]string
	Database       interfaces.Database
	Gateway        interfaces.FuncGateway
	QueueService   interfaces.QueueService
}

func NewOrchestrator(db interfaces.Database, gate interfaces.FuncGateway, queue interfaces.QueueService) *Orchestrator {
	return &Orchestrator{
		OrchestratorID: uuid.New().String(),
		Activated:	make(map[string]bool),
		Funcs    :  make(map[string]map[string]string),
		trigJob  :  make(map[string]string),
		Database:	db,
		Gateway :	gate,
		QueueService: queue,
	}
}

func (o *Orchestrator) ActivateService(ctx context.Context, funcID string) (bool, error) {
	   funcMetaData, err := o.Database.GetLambdaMetadata(ctx, funcID)
	   if err != nil {
		   return false, err
	   }
	   if funcMetaData["status"] != "compiled" {
		   return false, errors.New("Lambda function is not compiled")
	   }
	   _, err = o.Gateway.ActivateJob(funcID, funcMetaData["user_id"])
	   if err != nil {
		   return false, err
	   }
	   o.Funcs[funcID] = funcMetaData
	   o.Activated[funcID] = true
	   utils.Info("Activated service for funcID: " + funcID)
	   return true, nil
}

func (o *Orchestrator) DeactivateService(funcID string) (bool, error) {
	   _, err := o.Gateway.DeactivateJob(funcID, "")
	   if err != nil {
		   return false, err
	   }
	   delete(o.Activated, funcID)
	   delete(o.Funcs, funcID)
	   utils.Info("Deactivated service for funcID: " + funcID)
	   return true, nil
}

func (o* Orchestrator) ReceiveTrigger(ctx context.Context, trigID string, funcID string, input string) (bool, error) {
	   jobID := uuid.New().String()
	   metaData, exists := o.Funcs[funcID]
	   if !exists {
		   return false, errors.New("Function not activated")
	   }
	   o.trigJob[jobID] = trigID
	   err, ack := o.QueueService.Enqueue(ctx, jobID, funcID, metaData)
	   if err != nil {
		   utils.Error("Failed to enqueue job for funcID: " + funcID + ", trigID: " + trigID + ", error: " + err.Error())
		   return false, err
	   }
	   utils.Info("Trigger received and job enqueued for funcID: " + funcID + ", trigID: " + trigID + ", jobID: " + jobID)
	   return ack, nil
}


func (o *Orchestrator) IsServiceActivated(funcID string) bool {
	activated, exists := o.Activated[funcID]
	if !exists {
		return false
	}
	return activated
}