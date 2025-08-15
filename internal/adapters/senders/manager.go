package senders

import (
	"context"
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// Service manages sender configurations for script execution
type Service struct {
	configs map[string]*domain.SenderConfig
}

// NewService creates a new sender service
func NewService() *Service {
	return &Service{
		configs: make(map[string]*domain.SenderConfig),
	}
}

// LoadConfigs loads sender configurations from the environment
func (s *Service) LoadConfigs(configs map[string]*domain.SenderConfig) error {
	s.configs = configs
	return nil
}

// GetSender retrieves a sender configuration by name
func (s *Service) GetSender(name string) (*domain.SenderConfig, error) {
	// Handle special cases
	if name == "" {
		return s.getDefaultSender()
	}
	
	// Direct lookup
	if sender, ok := s.configs[name]; ok {
		return sender, nil
	}
	
	// Try case-insensitive lookup
	nameLower := strings.ToLower(name)
	for key, sender := range s.configs {
		if strings.ToLower(key) == nameLower {
			return sender, nil
		}
	}
	
	return nil, fmt.Errorf("sender '%s' not found", name)
}

// getDefaultSender returns the default sender configuration
func (s *Service) getDefaultSender() (*domain.SenderConfig, error) {
	// Check for explicitly marked default
	if sender, ok := s.configs["default"]; ok {
		return sender, nil
	}
	
	// If only one sender exists, use it
	if len(s.configs) == 1 {
		for _, sender := range s.configs {
			return sender, nil
		}
	}
	
	// Check for common default names
	for _, name := range []string{"local", "deployer", "dev"} {
		if sender, ok := s.configs[name]; ok {
			return sender, nil
		}
	}
	
	return nil, fmt.Errorf("no default sender configured")
}

// BuildEnvironment builds environment variables for a sender
func (s *Service) BuildEnvironment(ctx context.Context, senderName string) (map[string]string, error) {
	sender, err := s.GetSender(senderName)
	if err != nil {
		return nil, err
	}
	
	env := make(map[string]string)
	
	switch sender.Type {
	case "private_key":
		if sender.PrivateKey == "" {
			return nil, fmt.Errorf("private key not configured for sender %s", senderName)
		}
		env["SENDER_TYPE"] = "private_key"
		env["PRIVATE_KEY"] = sender.PrivateKey
		
	case "ledger":
		if sender.DerivationPath == "" {
			return nil, fmt.Errorf("derivation path not configured for ledger sender %s", senderName)
		}
		env["SENDER_TYPE"] = "ledger"
		env["DERIVATION_PATH"] = sender.DerivationPath
		
	case "safe":
		if sender.Safe == "" {
			return nil, fmt.Errorf("safe address not configured for sender %s", senderName)
		}
		env["SENDER_TYPE"] = "safe"
		env["SAFE_ADDRESS"] = sender.Safe
		
		// Add proposer configuration
		if sender.Proposer != nil {
			env["PROPOSER_TYPE"] = sender.Proposer.Type
			if sender.Proposer.Type == "private_key" {
				env["PROPOSER_PRIVATE_KEY"] = sender.Proposer.PrivateKey
			} else if sender.Proposer.Type == "ledger" {
				env["PROPOSER_DERIVATION_PATH"] = sender.Proposer.DerivationPath
			}
		}
		
	default:
		return nil, fmt.Errorf("unsupported sender type: %s", sender.Type)
	}
	
	// Add sender name for reference
	env["SENDER_NAME"] = senderName
	
	return env, nil
}

// ValidateSender validates a sender configuration
func (s *Service) ValidateSender(sender *domain.SenderConfig) error {
	if sender == nil {
		return fmt.Errorf("sender configuration is nil")
	}
	
	switch sender.Type {
	case "private_key":
		if sender.PrivateKey == "" {
			return fmt.Errorf("private key is required for private_key sender")
		}
		if !isValidPrivateKey(sender.PrivateKey) {
			return fmt.Errorf("invalid private key format")
		}
		
	case "ledger":
		if sender.DerivationPath == "" {
			return fmt.Errorf("derivation path is required for ledger sender")
		}
		if !isValidDerivationPath(sender.DerivationPath) {
			return fmt.Errorf("invalid derivation path format")
		}
		
	case "safe":
		if sender.Safe == "" {
			return fmt.Errorf("safe address is required for safe sender")
		}
		if !isValidAddress(sender.Safe) {
			return fmt.Errorf("invalid safe address format")
		}
		
		// Validate proposer if present
		if sender.Proposer != nil {
			if err := s.validateProposer(sender.Proposer); err != nil {
				return fmt.Errorf("invalid proposer configuration: %w", err)
			}
		}
		
	default:
		return fmt.Errorf("unknown sender type: %s", sender.Type)
	}
	
	return nil
}

// validateProposer validates a proposer configuration
func (s *Service) validateProposer(proposer *domain.ProposerConfig) error {
	switch proposer.Type {
	case "private_key":
		if proposer.PrivateKey == "" {
			return fmt.Errorf("private key is required for private_key proposer")
		}
		if !isValidPrivateKey(proposer.PrivateKey) {
			return fmt.Errorf("invalid private key format")
		}
		
	case "ledger":
		if proposer.DerivationPath == "" {
			return fmt.Errorf("derivation path is required for ledger proposer")
		}
		if !isValidDerivationPath(proposer.DerivationPath) {
			return fmt.Errorf("invalid derivation path format")
		}
		
	default:
		return fmt.Errorf("unknown proposer type: %s", proposer.Type)
	}
	
	return nil
}

// Helper functions for validation
func isValidPrivateKey(key string) bool {
	// Remove 0x prefix if present
	key = strings.TrimPrefix(key, "0x")
	
	// Check if it's 64 hex characters
	if len(key) != 64 {
		return false
	}
	
	// Check if all characters are hex
	for _, c := range key {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	
	return true
}

func isValidAddress(addr string) bool {
	// Remove 0x prefix if present
	addr = strings.TrimPrefix(addr, "0x")
	
	// Check if it's 40 hex characters
	if len(addr) != 40 {
		return false
	}
	
	// Check if all characters are hex
	for _, c := range addr {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	
	return true
}

func isValidDerivationPath(path string) bool {
	// Basic validation for BIP32 derivation path
	// Format: m/44'/60'/0'/0/0
	if !strings.HasPrefix(path, "m/") {
		return false
	}
	
	parts := strings.Split(path[2:], "/")
	if len(parts) < 2 {
		return false
	}
	
	for _, part := range parts {
		// Remove hardened marker if present
		part = strings.TrimSuffix(part, "'")
		
		// Check if it's a valid number
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	
	return true
}

// SenderResolver resolves sender configurations from various sources
type SenderResolver struct {
	service *Service
}

// NewSenderResolver creates a new sender resolver
func NewSenderResolver(service *Service) *SenderResolver {
	return &SenderResolver{
		service: service,
	}
}

// ResolveSender resolves a sender from name or environment
func (r *SenderResolver) ResolveSender(ctx context.Context, senderName string, env map[string]string) (*domain.SenderConfig, error) {
	// First try to get from service
	if senderName != "" {
		return r.service.GetSender(senderName)
	}
	
	// Try to build from environment variables
	if senderType, ok := env["SENDER_TYPE"]; ok {
		sender := &domain.SenderConfig{
			Type: senderType,
		}
		
		switch senderType {
		case "private_key":
			sender.PrivateKey = env["PRIVATE_KEY"]
		case "ledger":
			sender.DerivationPath = env["DERIVATION_PATH"]
		case "safe":
			sender.Safe = env["SAFE_ADDRESS"]
			// Build proposer if present
			if proposerType, ok := env["PROPOSER_TYPE"]; ok {
				sender.Proposer = &domain.ProposerConfig{
					Type: proposerType,
				}
				if proposerType == "private_key" {
					sender.Proposer.PrivateKey = env["PROPOSER_PRIVATE_KEY"]
				} else if proposerType == "ledger" {
					sender.Proposer.DerivationPath = env["PROPOSER_DERIVATION_PATH"]
				}
			}
		}
		
		// Validate before returning
		if err := r.service.ValidateSender(sender); err != nil {
			return nil, err
		}
		
		return sender, nil
	}
	
	// Fall back to default
	return r.service.getDefaultSender()
}