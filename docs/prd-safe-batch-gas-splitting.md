# PRD: Gas-Aware Safe Transaction Batch Splitting

## Problem

Currently, all Safe transactions within a single script execution are batched into a single MultiSend transaction. For scripts that deploy many contracts or execute many calls, this creates batches that can exceed the block gas limit, causing the Safe transaction to fail at execution time.

The failure happens late — after simulation succeeds, after the proposer signs, and after other signers approve. This is expensive in coordination cost and confusing to debug.

### Example

A deployment script that deploys 15 contracts through a Safe accumulates all 15 `CREATE` calls into a single MultiSend batch. The batch requires 25M gas but the chain's block gas limit is 30M. At 83% of the block limit, miners/validators may reject the transaction, or it may succeed only during low-congestion periods. A safer threshold would split this into two batches.

### Current Architecture

```
Simulation Phase (vm.prank):
  tx1 → globalQueue
  tx2 → globalQueue
  ...
  txN → globalQueue

Broadcast Phase:
  globalQueue → Safe.txQueue (all N transactions)
  Safe.broadcast() → single MultiSend(tx1...txN)
```

There is no gas tracking during simulation, no awareness of block gas limits, and no mechanism to split batches.

## Solution

Add gas metering during the simulation phase and gas-aware batch splitting during the broadcast phase.

### High-Level Flow

```
Simulation Phase (vm.prank):
  gasBefore = gasleft()
  tx1 → simulate → record gasUsed
  tx2 → simulate → record gasUsed
  ...
  txN → simulate → record gasUsed

Broadcast Phase:
  blockGasLimit = block.gaslimit (from fork RPC)
  threshold = blockGasLimit * 50 / 100

  batch = []
  batchGas = 0
  for each tx in Safe.txQueue:
    if batchGas + tx.gasUsed > threshold AND batch not empty:
      broadcastBatch(batch)
      batch = [tx]
      batchGas = tx.gasUsed
    else:
      batch.append(tx)
      batchGas += tx.gasUsed
  broadcastBatch(batch)  // final batch
```

### Design Decisions

**1. Gas measurement happens during simulation, splitting happens during broadcast**

The simulation phase already executes each transaction via `vm.prank`. We wrap each call with `gasleft()` before and after to measure gas consumed. This is cheap and accurate — the simulated execution environment mirrors the broadcast environment closely enough for gas estimation purposes.

The splitting logic lives in `GnosisSafeSender.broadcast()` where it has access to the accumulated queue and can emit the right events per batch.

**2. Block gas limit comes from `block.gaslimit`**

During simulation, the code runs against a fork. `block.gaslimit` on the fork reflects the actual chain's gas limit at the forked block. This is the simplest approach — no extra RPC calls, no configuration, and it automatically adapts to different chains (Ethereum 30M, Arbitrum 1.125B, etc).

**3. Threshold is 50% of block gas limit**

A 50% threshold provides a healthy safety margin:
- Leaves room for the MultiSend overhead and Safe.execTransaction wrapper gas
- Avoids competing with other transactions for block space
- Prevents near-limit edge cases where gas estimation inaccuracy causes reverts

This can be made configurable later if needed, but 50% is a sensible default.

**4. Gas overhead buffer for MultiSend wrapping**

The measured gas is per-transaction. The actual execution wraps transactions in MultiSend encoding and executes through `Safe.execTransaction`, which adds overhead. A fixed buffer (e.g. 100k gas) per batch accounts for:
- MultiSend calldata encoding
- Safe signature verification
- DelegateCall overhead

**5. A single transaction that exceeds the threshold goes through alone**

If a single transaction uses more than 50% of the block gas limit, it still gets its own batch rather than being rejected. The threshold controls when to *start a new batch*, not whether a transaction is allowed.

## Implementation

### Phase 1: Gas Metering in Simulation

#### 1.1 Add `gasUsed` to `SimulatedTransaction`

**File: `treb-sol/src/internal/types.sol`**

```solidity
struct SimulatedTransaction {
    bytes32 transactionId;
    bytes32 senderId;
    address sender;
    bytes returnData;
    Transaction transaction;
    uint256 gasUsed;          // NEW: gas consumed during simulation
}
```

#### 1.2 Measure gas during `Senders.simulate()`

**File: `treb-sol/src/internal/sender/Senders.sol`** — `simulate()` function

Wrap the `vm.prank` + `call` with `gasleft()` measurement:

```solidity
uint256 gasBefore = gasleft();

vm.prank(_sender.account);
(bool success, bytes memory returnData) =
    _transactions[i].to.call{value: _transactions[i].value}(_transactions[i].data);

uint256 gasAfter = gasleft();
uint256 gasUsed = gasBefore - gasAfter;
```

Set `gasUsed` on the `SimulatedTransaction`:

```solidity
SimulatedTransaction memory simulatedTx = SimulatedTransaction({
    transaction: _transactions[i],
    transactionId: transactionId,
    senderId: _sender.id,
    sender: _sender.account,
    returnData: returnData,
    gasUsed: gasUsed
});
```

### Phase 2: Gas-Aware Batch Splitting

#### 2.1 Split batches in `GnosisSafeSender.broadcast()`

**File: `treb-sol/src/internal/sender/GnosisSafeSender.sol`** — `broadcast()` function

Replace the current "collect all, broadcast once" logic with gas-aware chunking:

```solidity
function broadcast(Sender storage _sender) internal {
    if (_sender.txQueue.length == 0) return;

    uint256 gasThreshold = block.gaslimit * 50 / 100;
    uint256 BATCH_OVERHEAD = 100_000; // MultiSend + Safe.execTransaction overhead

    uint256 batchStart = 0;
    uint256 batchGas = BATCH_OVERHEAD;

    for (uint256 i = 0; i < _sender.txQueue.length; i++) {
        uint256 txGas = _sender.txQueue[i].gasUsed;

        // If adding this tx would exceed threshold, broadcast current batch first
        // (but only if current batch is non-empty)
        if (batchGas + txGas > gasThreshold && i > batchStart) {
            _broadcastBatch(_sender, batchStart, i);
            batchStart = i;
            batchGas = BATCH_OVERHEAD;
        }

        batchGas += txGas;
    }

    // Broadcast remaining batch
    _broadcastBatch(_sender, batchStart, _sender.txQueue.length);

    delete _sender.txQueue;
}
```

#### 2.2 Extract batch broadcasting to helper

Extract the current single-batch logic into `_broadcastBatch(sender, startIdx, endIdx)` that:
1. Builds `targets[]` and `datas[]` from the slice `[startIdx, endIdx)`
2. Validates no value transfers
3. Either executes directly (threshold-1) or proposes via API
4. Emits `SafeTransactionQueued`/`SafeTransactionExecuted` per batch

This means a single script run can produce *multiple* Safe transactions. Each batch gets its own `safeTxHash` and its own event.

### Phase 3: CLI Handling of Multiple Safe Transactions

#### 3.1 The Go CLI already supports multiple Safe transactions

The hydrator processes events, and multiple `SafeTransactionQueued` events will naturally produce multiple `SafeTransaction` entries in the result. No changes needed in the hydrator.

#### 3.2 Output rendering

When multiple batches are produced, the CLI should display each batch:

```
🔐 Safe Transactions:
  Batch 1/3: 5 transactions (est. 12.4M gas) → safeTxHash: 0xabc...
  Batch 2/3: 5 transactions (est. 13.1M gas) → safeTxHash: 0xdef...
  Batch 3/3: 3 transactions (est. 8.2M gas)  → safeTxHash: 0x123...
```

### Phase 4: Sequential Nonce Management

When splitting into multiple batches, each batch needs a sequential Safe nonce to ensure ordered execution. The current code calls `getNonce()` once per broadcast. With multiple batches, each subsequent batch must use `nonce + 1`, `nonce + 2`, etc.

The Safe API and direct execution both use `safeInstance.nonce()` to get the current nonce. For the first batch this is correct. For subsequent batches in the same script run:
- **API proposal path**: The Safe Transaction Service accepts sequential nonces — propose batch 1 with nonce N, batch 2 with nonce N+1, etc.
- **Direct execution path (threshold-1)**: After `execTransaction` succeeds, the Safe's on-chain nonce increments automatically, so `safeInstance.nonce()` returns the correct value for the next batch.

## Out of Scope

- **Configurable gas threshold**: Hardcode 50% for now. Can add `--gas-threshold` flag or treb.toml config later if needed.
- **Cross-sender gas awareness**: Only Safe batches are split. Private key transactions are already individual. Governor proposals are a separate concern.
- **Gas price estimation**: We only care about gas units, not gas cost. Price estimation is orthogonal.
- **Retry/resume for partial batch failures**: If batch 2 of 3 fails to propose, the script fails. The user can manually handle the remaining batches.

## Risks

1. **Gas measurement accuracy**: `gasleft()` in a forked environment may not perfectly match on-chain execution gas. The 50% threshold provides sufficient margin for this.
2. **MultiSend overhead scaling**: More transactions in a batch means more calldata encoding overhead. The fixed 100k buffer should cover typical cases but may need tuning for batches with very large calldata.
3. **Nonce gaps on partial failure**: If batch 1 is proposed but batch 2 fails, nonce N is queued but nonce N+1 is not. This is expected behavior — the user can re-run or manually propose the remaining transactions.
4. **Event ordering**: Multiple `SafeTransactionQueued` events per Safe sender is a new pattern. The CLI hydrator should handle this correctly since it processes events individually, but this needs verification.

## Testing Strategy

### Unit Tests (treb-sol)

1. **Gas measurement**: Verify `gasUsed` is populated on `SimulatedTransaction` after simulation
2. **Single batch (under threshold)**: 3 small transactions → 1 batch
3. **Multi batch (over threshold)**: 10 large transactions → 2-3 batches
4. **Single large transaction**: 1 transaction exceeding threshold → 1 batch (not rejected)
5. **Nonce sequencing**: Verify sequential nonces across batches

### Integration Tests (treb-cli)

1. **Small deployment through Safe**: All transactions fit in one batch (no behavioral change)
2. **Large deployment through Safe**: Transactions split into multiple batches, all events parsed correctly
3. **Output rendering**: Multiple batches displayed correctly in CLI output
