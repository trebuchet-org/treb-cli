package script

import (
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
)

// ConvertRawLogToTypesLog converts our internal RawLog format (from forge JSON output) to go-ethereum types.Log
func ConvertRawLogToTypesLog(rawLog RawLog) (*types.Log, error) {
	// Decode the hex data string to bytes
	data, err := hex.DecodeString(strings.TrimPrefix(rawLog.Data, "0x"))
	if err != nil {
		return nil, err
	}

	return &types.Log{
		Address: rawLog.Address,
		Topics:  rawLog.Topics,
		Data:    data,
		// These fields would be filled from context if available:
		BlockNumber: 0,
		TxHash:      common.Hash{},
		TxIndex:     0,
		BlockHash:   common.Hash{},
		Index:       0,
		Removed:     false,
	}, nil
}

// ConvertEventsLogToTypesLog converts our events.Log format to go-ethereum types.Log
func ConvertEventsLogToTypesLog(log events.Log) (*types.Log, error) {
	// Decode the hex data string to bytes
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, err
	}

	return &types.Log{
		Address: log.Address,
		Topics:  log.Topics,
		Data:    data,
		// These fields would be filled from context if available:
		BlockNumber: 0,
		TxHash:      common.Hash{},
		TxIndex:     0,
		BlockHash:   common.Hash{},
		Index:       0,
		Removed:     false,
	}, nil
}