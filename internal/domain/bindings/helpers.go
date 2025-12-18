package bindings

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

// GetEventID returns the event signature hash for a given event name
// This is a helper method that works alongside the generated ABI bindings
func (treb *Treb) GetEventID(eventName string) (common.Hash, error) {
	event, exists := treb.abi.Events[eventName]
	if !exists {
		return common.Hash{}, fmt.Errorf("event %s not found", eventName)
	}
	return event.ID, nil
}

// GetEventID returns the event signature hash for a given event name
// This is a helper method that works alongside the generated ABI bindings
func (createx *CreateX) GetEventID(eventName string) (common.Hash, error) {
	event, exists := createx.abi.Events[eventName]
	if !exists {
		return common.Hash{}, fmt.Errorf("event %s not found", eventName)
	}
	return event.ID, nil
}

func (e *TrebContractDeployed) String() string {
	return fmt.Sprintf(
		"%s: address=%s artifact=%s txid=%x",
		e.ContractEventName(),
		e.Location.String(),
		e.Deployment.Artifact,
		e.TransactionId,
	)
}

func (e *TrebTransactionSimulated) String() string {
	return fmt.Sprintf(
		"%s: txid=%x sender=%s to=%s",
		e.ContractEventName(),
		e.SimulatedTx.TransactionId,
		e.SimulatedTx.Sender,
		e.SimulatedTx.Transaction.To,
	)
}

func (e *TrebDeploymentCollision) String() string {
	return fmt.Sprintf(
		"%s: address=%s artifact=%s",
		e.ContractEventName(),
		e.ExistingContract.String(),
		e.DeploymentDetails.Artifact,
	)
}

func (e *TrebSafeTransactionExecuted) String() string {
	return fmt.Sprintf(
		"%s: safeTxHash=%x safe=%s sender=%s txids=%v",
		e.ContractEventName(),
		e.SafeTxHash,
		e.Safe.String(),
		e.Executor,
		lo.Map(e.TransactionIds, func(v [32]byte, i int) string {
			return fmt.Sprintf("%x", v)
		}),
	)
}

func (e *TrebSafeTransactionQueued) String() string {
	return fmt.Sprintf(
		"%s: safeTxHash=%x safe=%s sender=%s txids=%v",
		e.ContractEventName(),
		e.SafeTxHash,
		e.Safe.String(),
		e.Proposer,
		lo.Map(e.TransactionIds, func(v [32]byte, i int) string {
			return fmt.Sprintf("%x", v)
		}),
	)
}

func (e *TrebGovernorProposalCreated) String() string {
	return fmt.Sprintf(
		"%s: proposalId=%s governor=%s proposer=%s txids=%v",
		e.ContractEventName(),
		e.ProposalId.String(),
		e.Governor.String(),
		e.Proposer.String(),
		lo.Map(e.TransactionIds, func(v [32]byte, i int) string {
			return fmt.Sprintf("%x", v)
		}),
	)
}
