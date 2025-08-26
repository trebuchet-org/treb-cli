package config

import "github.com/ethereum/go-ethereum/common"

type SenderScriptConfig struct {
	UseLedger         bool
	UseTrezor         bool
	DerivationPaths   []string
	EncodedConfig     string
	SenderInitConfigs []SenderInitConfig
	Senders           []string
}

type SenderInitConfig struct {
	BaseConfig   SenderConfig
	Name         string
	Account      common.Address
	SenderType   [8]byte // bytes8 magic constant
	CanBroadcast bool
	Config       []byte // ABI-encoded config data
}

type SenderHWConfig struct {
	UseLedger       bool
	UseTrezor       bool
	DerivationPaths []string
}
