package script

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/events"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
)

// UpdateRegistryFromEvents updates the deployment registry with parsed events
func UpdateRegistryFromEvents(
	scriptEvents []ParsedEvent,
	networkName string,
	chainID uint64,
	namespace string,
	scriptPath string,
	broadcastPath string,
	indexer *contracts.Indexer,
) error {
	// Create script updater
	updater := registry.NewScriptUpdater(indexer)

	// Default namespace if not provided
	if namespace == "" {
		namespace = "default"
	}

	// Convert script events to events package events
	eventsPackageEvents := convertScriptEventsToEventsPackage(scriptEvents)

	// Build registry update from events
	registryUpdate := updater.BuildRegistryUpdate(
		eventsPackageEvents,
		namespace,
		chainID,
		networkName,
		scriptPath,
	)

	// Print summary for debugging
	fmt.Println("\n📊 Registry Update Summary:")
	fmt.Print(registryUpdate.GetSummary())

	// If dry run, save the update for inspection
	if registryUpdate.Metadata.DryRun {
		// Save to file for debugging
		data, _ := json.MarshalIndent(registryUpdate, "", "  ")
		if err := os.WriteFile("registry-update-dry-run.json", data, 0644); err != nil {
			PrintWarningMessage(fmt.Sprintf("Failed to save dry-run update: %v", err))
		} else {
			fmt.Println("\n💾 Dry-run registry update saved to: registry-update-dry-run.json")
		}
		return nil
	}

	// Enrich with broadcast data if available
	if broadcastPath != "" {
		enricher := registry.NewBroadcastEnricher()
		if err := enricher.EnrichFromBroadcastFile(registryUpdate, broadcastPath); err != nil {
			PrintWarningMessage(fmt.Sprintf("Failed to enrich from broadcast: %v", err))
			// Continue anyway - we can still save with partial data
		} else {
			fmt.Println("\n✨ Enriched registry update with broadcast data")
		}
	}

	// Create manager and apply update
	manager, err := registry.NewManager(".")
	if err != nil {
		return fmt.Errorf("failed to create registry manager: %w", err)
	}

	// Apply the update
	if err := registryUpdate.Apply(manager); err != nil {
		return fmt.Errorf("failed to apply registry update: %w", err)
	}

	PrintSuccessMessage(fmt.Sprintf("Updated registry for %s network in namespace %s", networkName, namespace))
	
	// Save applied update for debugging
	data, _ := json.MarshalIndent(registryUpdate, "", "  ")
	if err := os.WriteFile("registry-update-applied.json", data, 0644); err != nil {
		PrintWarningMessage(fmt.Sprintf("Failed to save applied update: %v", err))
	}

	return nil
}

// convertScriptEventsToEventsPackage converts script package events to events package events
func convertScriptEventsToEventsPackage(scriptEvents []ParsedEvent) []events.ParsedEvent {
	var eventsPackageEvents []events.ParsedEvent
	failedConversions := 0
	
	for _, scriptEvent := range scriptEvents {
		// The script package re-exports events package types, so we can cast directly
		if eventsEvent, ok := scriptEvent.(events.ParsedEvent); ok {
			eventsPackageEvents = append(eventsPackageEvents, eventsEvent)
		} else {
			failedConversions++
			fmt.Printf("⚠️ Failed to convert event type: %T\n", scriptEvent)
		}
	}
	
	if failedConversions > 0 {
		fmt.Printf("⚠️ Warning: %d events failed type conversion\n", failedConversions)
	}
	
	return eventsPackageEvents
}