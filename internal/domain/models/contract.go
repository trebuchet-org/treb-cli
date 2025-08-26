package models

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Contract represents information about a discovered contract
type Contract struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	ArtifactPath string    `json:"artifactPath,omitempty"`
	Version      string    `json:"version,omitempty"`
	Artifact     *Artifact `json:"artifact,omitempty"`
}

func (c *Contract) IsLibrary() bool {
	return strings.Contains(c.Source(), fmt.Sprintf("library %s", c.Name))
}

func (c *Contract) Source() string {
	data, err := os.ReadFile(c.Path)
	if err != nil {
		panic(err)
	}
	return string(data)
}

// BytecodeObject represents bytecode information in a Foundry artifact
type BytecodeObject struct {
	Object         string         `json:"object"`
	SourceMap      string         `json:"sourceMap"`
	LinkReferences map[string]any `json:"linkReferences"`
}

// Artifact represents a Foundry compilation artifact
type Artifact struct {
	ABI               json.RawMessage   `json:"abi"`
	Bytecode          BytecodeObject    `json:"bytecode"`
	DeployedBytecode  BytecodeObject    `json:"deployedBytecode"`
	MethodIdentifiers map[string]string `json:"methodIdentifiers"`
	RawMetadata       string            `json:"rawMetadata"`
	Metadata          ArtifactMetadata  `json:"metadata"`
}

// ArtifactMetadata represents the metadata section of a Foundry artifact
type ArtifactMetadata struct {
	Compiler struct {
		Version string `json:"version"`
	} `json:"compiler"`
	Language string `json:"language"`
	Output   struct {
		ABI      json.RawMessage `json:"abi"`
		DevDoc   json.RawMessage `json:"devdoc"`
		UserDoc  json.RawMessage `json:"userdoc"`
		Metadata string          `json:"metadata"`
	} `json:"output"`
	Settings struct {
		CompilationTarget map[string]string `json:"compilationTarget"`
	} `json:"settings"`
}
