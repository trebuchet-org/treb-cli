package domain

// BroadcastFile represents a Foundry broadcast file
type BroadcastFile struct {
	Chain        uint64                 `json:"chain"`
	Transactions []BroadcastTransaction `json:"transactions"`
	Receipts     []BroadcastReceipt     `json:"receipts"`
	Timestamp    uint64                 `json:"timestamp"`
	Commit       string                 `json:"commit"`
}

// BroadcastTransaction represents a transaction in a broadcast file
type BroadcastTransaction struct {
	Hash                string               `json:"hash"`
	Transaction         map[string]any       `json:"transaction"`
	ContractName        string               `json:"contractName"`
	ContractAddr        string               `json:"contractAddress"`
	Function            string               `json:"function"`
	Arguments           []any                `json:"arguments"`
	AdditionalContracts []AdditionalContract `json:"additionalContracts,omitempty"`
}

// AdditionalContract represents additional contracts deployed in a transaction
type AdditionalContract struct {
	ContractName string `json:"contractName"`
	ContractAddr string `json:"contractAddress"`
}

// BroadcastReceipt represents a receipt in a broadcast file
type BroadcastReceipt struct {
	TransactionHash string         `json:"transactionHash"`
	BlockNumber     string         `json:"blockNumber"`
	GasUsed         string         `json:"gasUsed"`
	Status          string         `json:"status"`
	ContractAddress string         `json:"contractAddress"`
	Logs            []BroadcastLog `json:"logs"`
}

// BroadcastLog represents a log entry from a transaction receipt
type BroadcastLog struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}
