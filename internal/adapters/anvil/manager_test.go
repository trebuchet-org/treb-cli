package anvil

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

func TestBuildAnvilArgs_Basic(t *testing.T) {
	instance := &domain.AnvilInstance{
		Port: "8545",
	}
	args := buildAnvilArgs(instance)
	assert.Equal(t, []string{"--port", "8545", "--host", "0.0.0.0"}, args)
}

func TestBuildAnvilArgs_WithChainID(t *testing.T) {
	instance := &domain.AnvilInstance{
		Port:    "9000",
		ChainID: "31337",
	}
	args := buildAnvilArgs(instance)
	assert.Equal(t, []string{"--port", "9000", "--host", "0.0.0.0", "--chain-id", "31337"}, args)
}

func TestBuildAnvilArgs_WithForkURL(t *testing.T) {
	instance := &domain.AnvilInstance{
		Port:    "9000",
		ForkURL: "https://rpc.sepolia.org",
	}
	args := buildAnvilArgs(instance)
	assert.Equal(t, []string{"--port", "9000", "--host", "0.0.0.0", "--fork-url", "https://rpc.sepolia.org"}, args)
}

func TestBuildAnvilArgs_WithChainIDAndForkURL(t *testing.T) {
	instance := &domain.AnvilInstance{
		Port:    "9000",
		ChainID: "11155111",
		ForkURL: "https://rpc.sepolia.org",
	}
	args := buildAnvilArgs(instance)
	assert.Equal(t, []string{
		"--port", "9000",
		"--host", "0.0.0.0",
		"--chain-id", "11155111",
		"--fork-url", "https://rpc.sepolia.org",
	}, args)
}

func TestBuildAnvilArgs_WithoutForkURL(t *testing.T) {
	instance := &domain.AnvilInstance{
		Port:    "8545",
		ChainID: "31337",
	}
	args := buildAnvilArgs(instance)
	// Should NOT contain --fork-url
	for _, arg := range args {
		assert.NotEqual(t, "--fork-url", arg)
	}
}

func TestSetFilePaths_DefaultInstance(t *testing.T) {
	m := NewManager()
	instance := &domain.AnvilInstance{}
	m.setFilePaths(instance)

	assert.Equal(t, "anvil", instance.Name)
	assert.Equal(t, DefaultAnvilPort, instance.Port)
	assert.Equal(t, "/tmp/treb-anvil-pid", instance.PidFile)
	assert.Equal(t, "/tmp/treb-anvil.log", instance.LogFile)
}

func TestSetFilePaths_NamedInstance(t *testing.T) {
	m := NewManager()
	instance := &domain.AnvilInstance{
		Name: "testnet",
		Port: "9000",
	}
	m.setFilePaths(instance)

	assert.Equal(t, "/tmp/treb-testnet.pid", instance.PidFile)
	assert.Equal(t, "/tmp/treb-testnet.log", instance.LogFile)
}

func TestSetFilePaths_ForkInstance(t *testing.T) {
	m := NewManager()
	instance := &domain.AnvilInstance{
		Name: "fork-sepolia",
		Port: "54321",
	}
	m.setFilePaths(instance)

	assert.Equal(t, "/tmp/treb-fork-sepolia.pid", instance.PidFile)
	assert.Equal(t, "/tmp/treb-fork-sepolia.log", instance.LogFile)
}

func TestSetFilePaths_PresetPathsPreserved(t *testing.T) {
	m := NewManager()
	instance := &domain.AnvilInstance{
		Name:    "fork-sepolia",
		Port:    "54321",
		PidFile: "/custom/path/my.pid",
		LogFile: "/custom/path/my.log",
	}
	m.setFilePaths(instance)

	assert.Equal(t, "/custom/path/my.pid", instance.PidFile)
	assert.Equal(t, "/custom/path/my.log", instance.LogFile)
}

// newMockRPCServer creates a test HTTP server that responds to JSON-RPC requests
func newMockRPCServer(t *testing.T, handler func(req rpcRequest) rpcResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode RPC request: %v", err)
		}
		resp := handler(req)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("failed to encode RPC response: %v", err)
		}
	}))
}

// instanceForServer creates an AnvilInstance pointing at the test server
func instanceForServer(t *testing.T, server *httptest.Server) *domain.AnvilInstance {
	t.Helper()
	// Extract port from server URL (format: http://127.0.0.1:<port>)
	parts := strings.Split(server.URL, ":")
	port := parts[len(parts)-1]
	return &domain.AnvilInstance{
		Name:    "test",
		Port:    port,
		PidFile: "/tmp/nonexistent-test.pid",
		LogFile: "/tmp/nonexistent-test.log",
	}
}

func TestTakeSnapshot_Success(t *testing.T) {
	server := newMockRPCServer(t, func(req rpcRequest) rpcResponse {
		assert.Equal(t, "evm_snapshot", req.Method)
		return rpcResponse{
			Jsonrpc: "2.0",
			Result:  "0x1",
			ID:      req.ID,
		}
	})
	defer server.Close()

	m := NewManager()
	instance := instanceForServer(t, server)

	snapshotID, err := m.TakeSnapshot(context.Background(), instance)
	require.NoError(t, err)
	assert.Equal(t, "0x1", snapshotID)
}

func TestTakeSnapshot_RPCError(t *testing.T) {
	server := newMockRPCServer(t, func(req rpcRequest) rpcResponse {
		return rpcResponse{
			Jsonrpc: "2.0",
			Error:   &rpcError{Code: -32000, Message: "snapshot failed"},
			ID:      req.ID,
		}
	})
	defer server.Close()

	m := NewManager()
	instance := instanceForServer(t, server)

	_, err := m.TakeSnapshot(context.Background(), instance)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "snapshot failed")
}

func TestRevertSnapshot_Success(t *testing.T) {
	server := newMockRPCServer(t, func(req rpcRequest) rpcResponse {
		assert.Equal(t, "evm_revert", req.Method)
		require.Len(t, req.Params, 1)
		assert.Equal(t, "0x1", req.Params[0])
		return rpcResponse{
			Jsonrpc: "2.0",
			Result:  true,
			ID:      req.ID,
		}
	})
	defer server.Close()

	m := NewManager()
	instance := instanceForServer(t, server)

	err := m.RevertSnapshot(context.Background(), instance, "0x1")
	require.NoError(t, err)
}

func TestRevertSnapshot_ReturnsFalse(t *testing.T) {
	server := newMockRPCServer(t, func(req rpcRequest) rpcResponse {
		return rpcResponse{
			Jsonrpc: "2.0",
			Result:  false,
			ID:      req.ID,
		}
	})
	defer server.Close()

	m := NewManager()
	instance := instanceForServer(t, server)

	err := m.RevertSnapshot(context.Background(), instance, "0xbad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evm_revert returned false")
}

func TestRevertSnapshot_RPCError(t *testing.T) {
	server := newMockRPCServer(t, func(req rpcRequest) rpcResponse {
		return rpcResponse{
			Jsonrpc: "2.0",
			Error:   &rpcError{Code: -32000, Message: "revert failed"},
			ID:      req.ID,
		}
	})
	defer server.Close()

	m := NewManager()
	instance := instanceForServer(t, server)

	err := m.RevertSnapshot(context.Background(), instance, "0x1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "revert failed")
}
