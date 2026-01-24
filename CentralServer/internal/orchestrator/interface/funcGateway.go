package interfaces

type FuncGateway interface {
	ActivateJob(funcID string, userID string) (bool, error)
	DeactivateJob(funcID string, userID string) (bool, error)
}
