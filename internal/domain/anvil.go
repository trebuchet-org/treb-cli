package domain

// AnvilInstance represents a local anvil node instance
type AnvilInstance struct {
	Name    string `json:"name"`
	Port    string `json:"port"`
	ChainID string `json:"chainId,omitempty"`
	ForkURL string `json:"forkUrl,omitempty"`
	PidFile string `json:"pidFile"`
	LogFile string `json:"logFile"`
}

// AnvilStatus represents the status of an anvil instance
type AnvilStatus struct {
	Running         bool   `json:"running"`
	PID             int    `json:"pid,omitempty"`
	RPCURL          string `json:"rpcUrl,omitempty"`
	LogFile         string `json:"logFile"`
	RPCHealthy      bool   `json:"rpcHealthy"`
	CreateXDeployed bool   `json:"createXDeployed"`
	CreateXAddress  string `json:"createXAddress,omitempty"`
	Error           string `json:"error,omitempty"`
}
