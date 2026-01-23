package interfaces
type RunTrigger interface {
	TriggerRun(lambdaID string) error
}
