package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Service handles contract verification across different explorers
type Service struct {
	client       *http.Client
	apiKeys      map[string]string
	explorerURLs map[string]string
}

// NewService creates a new verification service
func NewService() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKeys:      make(map[string]string),
		explorerURLs: make(map[string]string),
	}
}

// SetAPIKey sets the API key for a specific network
func (s *Service) SetAPIKey(network string, apiKey string) {
	s.apiKeys[network] = apiKey
}

// SetExplorerURL sets the explorer URL for a specific network
func (s *Service) SetExplorerURL(network string, url string) {
	s.explorerURLs[network] = url
}

// VerifyContract verifies a contract on the blockchain explorer
func (s *Service) VerifyContract(ctx context.Context, params VerificationParams) (*VerificationResult, error) {
	// Determine explorer type based on network
	explorerType := s.getExplorerType(params.Network)

	switch explorerType {
	case "etherscan":
		return s.verifyOnEtherscan(ctx, params)
	case "blockscout":
		return s.verifyOnBlockscout(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported explorer type for network %s", params.Network)
	}
}

// VerificationParams contains parameters for contract verification
type VerificationParams struct {
	Network          string
	Address          string
	ContractName     string
	SourceCode       string
	CompilerVersion  string
	Optimization     bool
	OptimizationRuns int
	ConstructorArgs  string
	Libraries        map[string]string
	EVMVersion       string
}

// VerificationResult contains the result of verification
type VerificationResult struct {
	Success        bool
	Message        string
	ExplorerURL    string
	VerificationID string
}

// getExplorerType determines the explorer type for a network
func (s *Service) getExplorerType(network string) string {
	// Most EVM chains use Etherscan-compatible APIs
	blockscoutNetworks := []string{"gnosis", "xdai", "sokol"}

	for _, bn := range blockscoutNetworks {
		if strings.EqualFold(network, bn) {
			return "blockscout"
		}
	}

	return "etherscan"
}

// verifyOnEtherscan verifies a contract using Etherscan API
func (s *Service) verifyOnEtherscan(ctx context.Context, params VerificationParams) (*VerificationResult, error) {
	apiKey, ok := s.apiKeys[params.Network]
	if !ok {
		return nil, fmt.Errorf("no API key configured for network %s", params.Network)
	}

	explorerURL, ok := s.explorerURLs[params.Network]
	if !ok {
		return nil, fmt.Errorf("no explorer URL configured for network %s", params.Network)
	}

	// Build verification request
	data := url.Values{}
	data.Set("apikey", apiKey)
	data.Set("module", "contract")
	data.Set("action", "verifysourcecode")
	data.Set("contractaddress", params.Address)
	data.Set("sourceCode", params.SourceCode)
	data.Set("codeformat", "solidity-single-file")
	data.Set("contractname", params.ContractName)
	data.Set("compilerversion", params.CompilerVersion)
	data.Set("optimizationUsed", boolToString(params.Optimization))
	if params.Optimization {
		data.Set("runs", fmt.Sprintf("%d", params.OptimizationRuns))
	}
	if params.ConstructorArgs != "" {
		data.Set("constructorArguements", params.ConstructorArgs) // Note: Etherscan typo
	}
	if params.EVMVersion != "" {
		data.Set("evmversion", params.EVMVersion)
	}

	// Handle libraries
	if len(params.Libraries) > 0 {
		libString := s.formatLibraries(params.Libraries)
		data.Set("libraryname", libString)
	}

	// Submit verification
	resp, err := s.client.PostForm(explorerURL+"/api", data)
	if err != nil {
		return nil, fmt.Errorf("failed to submit verification: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result etherscanResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != "1" {
		return &VerificationResult{
			Success: false,
			Message: result.Result,
		}, nil
	}

	// Return success with GUID for status checking
	return &VerificationResult{
		Success:        true,
		Message:        "Verification submitted successfully",
		VerificationID: result.Result,
		ExplorerURL:    fmt.Sprintf("%s/address/%s#code", explorerURL, params.Address),
	}, nil
}

// verifyOnBlockscout verifies a contract using Blockscout API
func (s *Service) verifyOnBlockscout(ctx context.Context, params VerificationParams) (*VerificationResult, error) {
	explorerURL, ok := s.explorerURLs[params.Network]
	if !ok {
		return nil, fmt.Errorf("no explorer URL configured for network %s", params.Network)
	}

	// Build verification request for Blockscout
	payload := map[string]interface{}{
		"addressHash":          params.Address,
		"name":                 params.ContractName,
		"compilerVersion":      params.CompilerVersion,
		"optimization":         params.Optimization,
		"optimizationRuns":     params.OptimizationRuns,
		"contractSourceCode":   params.SourceCode,
		"constructorArguments": params.ConstructorArgs,
		"evmVersion":           params.EVMVersion,
		"libraries":            params.Libraries,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", explorerURL+"/api/v1/verified_smart_contracts", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req) //nolint:gosec // URL is constructed from configured explorer endpoint
	if err != nil {
		return nil, fmt.Errorf("failed to submit verification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &VerificationResult{
			Success: false,
			Message: fmt.Sprintf("verification failed: %s", string(body)),
		}, nil
	}

	return &VerificationResult{
		Success:     true,
		Message:     "Contract verified successfully",
		ExplorerURL: fmt.Sprintf("%s/address/%s", explorerURL, params.Address),
	}, nil
}

// CheckVerificationStatus checks the status of a pending verification
func (s *Service) CheckVerificationStatus(ctx context.Context, network string, guid string) (*VerificationResult, error) {
	apiKey, ok := s.apiKeys[network]
	if !ok {
		return nil, fmt.Errorf("no API key configured for network %s", network)
	}

	explorerURL, ok := s.explorerURLs[network]
	if !ok {
		return nil, fmt.Errorf("no explorer URL configured for network %s", network)
	}

	// Build status check request
	params := url.Values{}
	params.Set("apikey", apiKey)
	params.Set("module", "contract")
	params.Set("action", "checkverifystatus")
	params.Set("guid", guid)

	resp, err := s.client.Get(explorerURL + "/api?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to check status: %w", err)
	}
	defer resp.Body.Close()

	var result etherscanResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check if still pending
	if strings.Contains(strings.ToLower(result.Result), "pending") {
		return &VerificationResult{
			Success: false,
			Message: "Verification still pending",
		}, nil
	}

	// Check if failed
	if result.Status != "1" {
		return &VerificationResult{
			Success: false,
			Message: result.Result,
		}, nil
	}

	return &VerificationResult{
		Success: true,
		Message: "Contract verified successfully",
	}, nil
}

// formatLibraries formats library addresses for Etherscan
func (s *Service) formatLibraries(libraries map[string]string) string {
	var parts []string
	for name, address := range libraries {
		parts = append(parts, fmt.Sprintf("%s:%s", name, address))
	}
	return strings.Join(parts, ",")
}

// etherscanResponse represents Etherscan API response
type etherscanResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

// boolToString converts bool to "0" or "1" for Etherscan API
func boolToString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// MultiVerifier handles verification across multiple explorers
type MultiVerifier struct {
	services map[string]*Service
}

// NewMultiVerifier creates a new multi-explorer verifier
func NewMultiVerifier() *MultiVerifier {
	return &MultiVerifier{
		services: make(map[string]*Service),
	}
}

// AddService adds a verification service for an explorer
func (m *MultiVerifier) AddService(name string, service *Service) {
	m.services[name] = service
}

// VerifyOnAll attempts to verify on all configured explorers
func (m *MultiVerifier) VerifyOnAll(ctx context.Context, params VerificationParams) map[string]*VerificationResult {
	results := make(map[string]*VerificationResult)

	for name, service := range m.services {
		result, err := service.VerifyContract(ctx, params)
		if err != nil {
			results[name] = &VerificationResult{
				Success: false,
				Message: err.Error(),
			}
		} else {
			results[name] = result
		}
	}

	return results
}
