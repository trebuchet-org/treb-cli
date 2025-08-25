package abi

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// EventDecoder decodes arbitrary events emitted during execution
type EventDecoder struct {
	abiResolver usecase.ABIResolver
	log         *slog.Logger
}

type LogEntry forge.LogEntry

// NewTransactionDecoder creates a new transaction decoder
func NewEventDecoder(abiResolver usecase.ABIResolver, log *slog.Logger) *EventDecoder {
	return &EventDecoder{
		abiResolver: abiResolver,
		log:         log.With("component", "TxDecoder"),
	}
}

func (l *LogEntry) IsDecoded() bool {
	// Sometimes Decoded has name but no params.
	return (l.Decoded.Name != "" && (len(l.Decoded.Params) > 0 || (len(l.RawLog.Topics) == 1 && l.RawLog.Data == "")))
}

func (l *LogEntry) RawData() ([]byte, error) {
	// Combine all the topics and data into a single byte array
	// Each topic is 32 bytes, and we concatenate them with the data
	result := make([]byte, 0, len(l.RawLog.Topics)*32+len(l.RawLog.Data))

	// Add each topic (they are common.Hash which is [32]byte)
	for _, topic := range l.RawLog.Topics {
		result = append(result, topic.Bytes()...)
	}

	data, err := hex.DecodeString(l.RawLog.Data)
	if err != nil {
		return nil, err
	}
	// Add the data
	result = append(result, data...)

	return result, nil
}

func (e *EventDecoder) DecodeEvent(log *forge.LogEntry, emitter common.Address) (*forge.LogEntry, error) {
	wrappedLog := (*LogEntry)(log)
	if wrappedLog.IsDecoded() {
		e.log.Debug("Log entry already decoded, skipping")
		return log, nil
	}
	// Try to find ABI for the emitter address
	abi, err := e.abiResolver.FindByAddress(context.Background(), emitter)
	if err != nil {
		return log, err
	}
	return e.DecodeEventFromABI(log, abi)
}

func (e *EventDecoder) DecodeEventFromABI(log *forge.LogEntry, abi *abi.ABI) (*forge.LogEntry, error) {
	wrappedLog := (*LogEntry)(log)
	if wrappedLog.IsDecoded() {
		e.log.Debug("Log entry already decoded, skipping")
		return log, nil
	} else {
		decodedLog, err := e.decodeRawLog(wrappedLog, abi)
		if err != nil {
			return log, err
		}
		return (*forge.LogEntry)(decodedLog), nil
	}
}

func (e *EventDecoder) decodeRawLog(log *LogEntry, abi *abi.ABI) (*LogEntry, error) {
	// use the abi resolver to resolve as much as possible from the logEntry and return a new LogEntry with
	// Decoded filled in

	// If there are no topics, we can't decode
	if len(log.RawLog.Topics) == 0 {
		return log, nil
	}

	// Get the event signature (first topic)
	eventSig := log.RawLog.Topics[0]

	// Find the event in the ABI that matches the signature
	for _, event := range abi.Events {
		if event.ID == eventSig {
			// Decode the event
			decodedParams := make(map[string]any)

			// First decode indexed parameters from topics
			if len(log.RawLog.Topics) > 1 {
				// Create a list of indexed inputs only
				var indexedInputs ethabi.Arguments
				for _, input := range event.Inputs {
					if input.Indexed {
						indexedInputs = append(indexedInputs, input)
					}
				}

				// Skip the first topic (event signature)
				err := ethabi.ParseTopicsIntoMap(decodedParams, indexedInputs, log.RawLog.Topics[1:])
				if err != nil {
					return nil, fmt.Errorf("failed to parse topics: %w", err)
				}
			}

			// Then decode non-indexed parameters from data
			if log.RawLog.Data != "" {
				data, err := hex.DecodeString(strings.TrimPrefix(log.RawLog.Data, "0x"))
				if err != nil {
					return nil, fmt.Errorf("failed to decode hex data: %w", err)
				}

				// Create a sub-map for non-indexed inputs only
				var nonIndexedInputs ethabi.Arguments
				for _, input := range event.Inputs {
					if !input.Indexed {
						nonIndexedInputs = append(nonIndexedInputs, input)
					}
				}

				if len(nonIndexedInputs) > 0 && len(data) > 0 {
					values, err := nonIndexedInputs.Unpack(data)
					if err != nil {
						return nil, fmt.Errorf("failed to unpack event data: %w", err)
					}

					// Map the unpacked values to parameter names
					for i, input := range nonIndexedInputs {
						if i < len(values) {
							decodedParams[input.Name] = values[i]
						}
					}
				}
			}

			// Convert decoded params to forge.DecodedParam format
			var params [][]string
			for _, input := range event.Inputs {
				if val, ok := decodedParams[input.Name]; ok {
					params = append(params, []string{input.Name, FormatValue(val, input.Type.String())})
				}
			}

			// Create a new log entry with decoded data
			decodedLog := *log
			decodedLog.Decoded = forge.DecodedLog{
				Name:   event.RawName,
				Params: params,
			}

			return &decodedLog, nil
		}
	}

	// If we couldn't decode with a specific ABI, just return the original
	return log, nil
}
