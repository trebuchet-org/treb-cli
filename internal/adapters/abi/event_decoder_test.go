package abi

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// MockABIResolver is a mock implementation of ABIResolver
type MockABIResolver struct {
	mock.Mock
}

func (m *MockABIResolver) Get(ctx context.Context, artifact *models.Artifact) (*abi.ABI, error) {
	args := m.Called(ctx, artifact)
	if abiVal := args.Get(0); abiVal != nil {
		return abiVal.(*abi.ABI), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockABIResolver) FindByRef(ctx context.Context, contractRef string) (*abi.ABI, error) {
	args := m.Called(ctx, contractRef)
	if abiVal := args.Get(0); abiVal != nil {
		return abiVal.(*abi.ABI), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockABIResolver) FindByAddress(ctx context.Context, address common.Address) (*abi.ABI, error) {
	args := m.Called(ctx, address)
	if abiVal := args.Get(0); abiVal != nil {
		return abiVal.(*abi.ABI), args.Error(1)
	}
	return nil, args.Error(1)
}

// MockDeploymentRepository is a mock implementation
type MockDeploymentRepository struct {
	usecase.DeploymentRepository // Embed to satisfy interface
	mock.Mock
}

func (m *MockDeploymentRepository) GetDeploymentByAddress(ctx context.Context, chainID uint64, address string) (*models.Deployment, error) {
	args := m.Called(ctx, chainID, address)
	if dep := args.Get(0); dep != nil {
		return dep.(*models.Deployment), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestEventDecoder_DecodeEvent(t *testing.T) {
	tests := []struct {
		name        string
		log         *forge.LogEntry
		emitter     common.Address
		mockSetup   func(*MockABIResolver)
		wantDecoded bool
		wantName    string
		wantParams  int
	}{
		{
			name: "Already decoded event",
			log: &forge.LogEntry{
				Decoded: forge.DecodedLog{
					Name: "Transfer",
					Params: [][]string{
						{"from", "0x1234..."},
						{"to", "0x5678..."},
						{"value", "1000"},
					},
				},
				RawLog: forge.RawLogEntry{
					Topics: []common.Hash{
						common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"), // Transfer event sig
					},
					Data: "",
				},
			},
			emitter:     common.HexToAddress("0x1234567890123456789012345678901234567890"),
			wantDecoded: true,
			wantName:    "Transfer",
			wantParams:  3,
		},
		{
			name: "Decode Transfer event",
			log: &forge.LogEntry{
				RawLog: forge.RawLogEntry{
					Topics: []common.Hash{
						common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"), // Transfer event sig
						common.HexToHash("0x000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266"), // from (indexed)
						common.HexToHash("0x00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c8"), // to (indexed)
					},
					Data: "00000000000000000000000000000000000000000000000000000000000003e8", // value = 1000
				},
			},
			emitter: common.HexToAddress("0x1234567890123456789012345678901234567890"),
			mockSetup: func(m *MockABIResolver) {
				// Create a real ABI with Transfer event
				abiJSON := `[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
				contractABI, _ := abi.JSON(strings.NewReader(abiJSON))

				m.On("FindByAddress", mock.Anything, mock.Anything).Return(&contractABI, nil)
			},
			wantDecoded: true,
			wantName:    "Transfer",
			wantParams:  3,
		},
		{
			name: "No topics - cannot decode",
			log: &forge.LogEntry{
				RawLog: forge.RawLogEntry{
					Topics: []common.Hash{},
					Data:   "",
				},
			},
			emitter:     common.HexToAddress("0x1234567890123456789012345678901234567890"),
			wantDecoded: false,
		},
		{
			name: "ABI not found",
			log: &forge.LogEntry{
				RawLog: forge.RawLogEntry{
					Topics: []common.Hash{
						common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
					},
					Data: "",
				},
			},
			emitter: common.HexToAddress("0x1234567890123456789012345678901234567890"),
			mockSetup: func(m *MockABIResolver) {
				m.On("FindByAddress", mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
			wantDecoded: false,
			// This test expects an error since the decoder returns an error when ABI is not found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockResolver := new(MockABIResolver)
			mockRepo := new(MockDeploymentRepository)

			if tt.mockSetup != nil {
				tt.mockSetup(mockResolver)
			}

			// Create decoder
			decoder := NewEventDecoder(
				mockResolver,
				mockRepo,
				&forge.HydratedRunResult{},
				slog.Default(),
			)

			// Decode event
			result, err := decoder.DecodeEvent(tt.log, tt.emitter)
			assert.NoError(t, err)

			// Verify results
			if tt.wantDecoded {
				assert.NotEmpty(t, result.Decoded.Name, "Expected decoded event name")
				assert.Equal(t, tt.wantName, result.Decoded.Name)
				assert.Len(t, result.Decoded.Params, tt.wantParams)
			} else {
				assert.Empty(t, result.Decoded.Name, "Expected no decoded event")
			}

			mockResolver.AssertExpectations(t)
		})
	}
}

func TestLogEntry_RawData(t *testing.T) {
	tests := []struct {
		name        string
		log         LogEntry
		expected    []byte
		expectError bool
	}{
		{
			name: "Combine topics and data",
			log: LogEntry{
				RawLog: forge.RawLogEntry{
					Topics: []common.Hash{
						common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
						common.HexToHash("0x000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266"),
						common.HexToHash("0x00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c8"),
					},
					Data: "00000000000000000000000000000000000000000000000000000000000003e8",
				},
			},
			expected: func() []byte {
				result := make([]byte, 0, 3*32+32) // 3 topics + 1 data chunk
				result = append(result, common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef").Bytes()...)
				result = append(result, common.HexToHash("0x000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266").Bytes()...)
				result = append(result, common.HexToHash("0x00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c8").Bytes()...)
				result = append(result, common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000003e8")...)
				return result
			}(),
		},
		{
			name: "Only topics, no data",
			log: LogEntry{
				RawLog: forge.RawLogEntry{
					Topics: []common.Hash{
						common.HexToHash("0xaabbccdd"),
					},
					Data: "",
				},
			},
			expected: common.HexToHash("0xaabbccdd").Bytes(),
		},
		{
			name: "No topics, only data",
			log: LogEntry{
				RawLog: forge.RawLogEntry{
					Topics: []common.Hash{},
					Data:   "01020304",
				},
			},
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name: "Invalid hex data",
			log: LogEntry{
				RawLog: forge.RawLogEntry{
					Topics: []common.Hash{},
					Data:   "invalid-hex",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.log.RawData()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestLogEntry_IsDecoded(t *testing.T) {
	tests := []struct {
		name     string
		log      LogEntry
		expected bool
	}{
		{
			name: "Decoded with params",
			log: LogEntry{
				Decoded: forge.DecodedLog{
					Name: "Transfer",
					Params: [][]string{
						{"from", "0x123"},
						{"to", "0x456"},
					},
				},
			},
			expected: true,
		},
		{
			name: "Decoded with name but no params - single topic and no data",
			log: LogEntry{
				Decoded: forge.DecodedLog{
					Name:   "Sync",
					Params: [][]string{},
				},
				RawLog: forge.RawLogEntry{
					Topics: []common.Hash{common.HexToHash("0x123")},
					Data:   "",
				},
			},
			expected: true,
		},
		{
			name: "Not decoded - empty name",
			log: LogEntry{
				Decoded: forge.DecodedLog{
					Name:   "",
					Params: [][]string{},
				},
			},
			expected: false,
		},
		{
			name: "Not decoded - has name but multiple topics",
			log: LogEntry{
				Decoded: forge.DecodedLog{
					Name:   "Transfer",
					Params: [][]string{},
				},
				RawLog: forge.RawLogEntry{
					Topics: []common.Hash{
						common.HexToHash("0x123"),
						common.HexToHash("0x456"),
					},
					Data: "",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.log.IsDecoded()
			assert.Equal(t, tt.expected, result)
		})
	}
}

