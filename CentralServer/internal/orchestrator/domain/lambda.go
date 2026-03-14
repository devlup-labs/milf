package domain

type Lambda struct {
	FuncID string `json:"func_id"`
	MetaData map[string]string `json:"meta_data"`
	UserID string `json:"user_id"`
	CompileStatus string `json:"compile_status"`
} 

//this is not being used as not needed, simple chekcs if the function is triggered or not is in the orchestrator.go file only



