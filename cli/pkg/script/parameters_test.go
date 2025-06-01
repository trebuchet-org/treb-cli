package script

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParameterParser_ParseCustomEnvString(t *testing.T) {
	parser := NewParameterParser()

	tests := []struct {
		name     string
		input    string
		expected []Parameter
	}{
		{
			name:  "basic parameters",
			input: "{string} label Label for the proxy{address} owner Owner address{uint256} amount Token amount",
			expected: []Parameter{
				{Name: "label", Type: TypeString, Description: "Label for the proxy", Optional: false},
				{Name: "owner", Type: TypeAddress, Description: "Owner address", Optional: false},
				{Name: "amount", Type: TypeUint256, Description: "Token amount", Optional: false},
			},
		},
		{
			name:  "with optional parameters",
			input: "{string:optional} description Optional description{address} owner Owner address",
			expected: []Parameter{
				{Name: "description", Type: TypeString, Description: "Optional description", Optional: true},
				{Name: "owner", Type: TypeAddress, Description: "Owner address", Optional: false},
			},
		},
		{
			name:  "meta types",
			input: "{sender} deployer The sender to use{deployment} impl Implementation{artifact} token Token artifact",
			expected: []Parameter{
				{Name: "deployer", Type: TypeSender, Description: "The sender to use", Optional: false},
				{Name: "impl", Type: TypeDeployment, Description: "Implementation", Optional: false},
				{Name: "token", Type: TypeArtifact, Description: "Token artifact", Optional: false},
			},
		},
		{
			name:  "real example from DeployUCProxy",
			input: "{string} label Label for the proxy and implementation{sender} deployer The sender which will deploy the contract{deployment} implementation Implementation to use for the proxy{artifact} implArtifact The implementation artifact to deploy{artifact} proxyArtifact The proxy artifact to deploy",
			expected: []Parameter{
				{Name: "label", Type: TypeString, Description: "Label for the proxy and implementation", Optional: false},
				{Name: "deployer", Type: TypeSender, Description: "The sender which will deploy the contract", Optional: false},
				{Name: "implementation", Type: TypeDeployment, Description: "Implementation to use for the proxy", Optional: false},
				{Name: "implArtifact", Type: TypeArtifact, Description: "The implementation artifact to deploy", Optional: false},
				{Name: "proxyArtifact", Type: TypeArtifact, Description: "The proxy artifact to deploy", Optional: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parseCustomEnvString(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParameterParser_ValidateValue(t *testing.T) {
	parser := NewParameterParser()

	tests := []struct {
		name      string
		param     Parameter
		value     string
		expectErr bool
	}{
		// String type
		{name: "valid string", param: Parameter{Type: TypeString}, value: "hello", expectErr: false},
		{name: "empty string optional", param: Parameter{Type: TypeString, Optional: true}, value: "", expectErr: false},
		{name: "empty string required", param: Parameter{Type: TypeString, Optional: false}, value: "", expectErr: true},

		// Address type
		{name: "valid address", param: Parameter{Type: TypeAddress}, value: "0x1234567890123456789012345678901234567890", expectErr: false},
		{name: "invalid address short", param: Parameter{Type: TypeAddress}, value: "0x123", expectErr: true},
		{name: "invalid address no prefix", param: Parameter{Type: TypeAddress}, value: "1234567890123456789012345678901234567890", expectErr: true},

		// Uint256 type
		{name: "valid uint decimal", param: Parameter{Type: TypeUint256}, value: "12345", expectErr: false},
		{name: "valid uint hex", param: Parameter{Type: TypeUint256}, value: "0x1234", expectErr: false},
		{name: "invalid uint", param: Parameter{Type: TypeUint256}, value: "not-a-number", expectErr: true},

		// Int256 type
		{name: "valid int positive", param: Parameter{Type: TypeInt256}, value: "12345", expectErr: false},
		{name: "valid int negative", param: Parameter{Type: TypeInt256}, value: "-12345", expectErr: false},
		{name: "valid int hex", param: Parameter{Type: TypeInt256}, value: "0x1234", expectErr: false},

		// Bytes32 type
		{name: "valid bytes32", param: Parameter{Type: TypeBytes32}, value: "0x0000000000000000000000000000000000000000000000000000000000000000", expectErr: false},
		{name: "invalid bytes32 short", param: Parameter{Type: TypeBytes32}, value: "0x1234", expectErr: true},

		// Bytes type
		{name: "valid bytes", param: Parameter{Type: TypeBytes}, value: "0x1234", expectErr: false},
		{name: "invalid bytes odd length", param: Parameter{Type: TypeBytes}, value: "0x123", expectErr: true},
		{name: "invalid bytes no prefix", param: Parameter{Type: TypeBytes}, value: "1234", expectErr: true},

		// Meta types (no validation here)
		{name: "sender", param: Parameter{Type: TypeSender}, value: "deployer", expectErr: false},
		{name: "deployment", param: Parameter{Type: TypeDeployment}, value: "Counter:v1", expectErr: false},
		{name: "artifact", param: Parameter{Type: TypeArtifact}, value: "MyToken", expectErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidateValue(tt.param, tt.value)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}