package model

type Snapshot struct {
	ProjectPath string   `json:"projectPath"`
	Detected    []string `json:"detected"` 
	Files       []string `json:"files"`    
}