package domain 

type Trigger struct {
	TrigID string `json:"trig_id"`
	FuncID string `json:"func_id"`
	UserID string `json:"user_id"`
	Input  string `json:"input"`
}

//this is not being used as not needed, simple chekcs if the function is triggered or not is in the orchestrator.go file only