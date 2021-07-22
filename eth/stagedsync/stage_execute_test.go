package stagedsync

import (
	"context"
	"testing"

	"github.com/ledgerwatch/erigon/common/changeset"
	"github.com/ledgerwatch/erigon/common/dbutils"
	"github.com/ledgerwatch/erigon/eth/stagedsync/stages"
	"github.com/ledgerwatch/erigon/ethdb/kv"
	"github.com/ledgerwatch/erigon/ethdb/prune"
	"github.com/stretchr/testify/assert"
)

func TestUnwindExecutionStagePlainStatic(t *testing.T) {
	ctx, assert := context.Background(), assert.New(t)
	_, tx1 := kv.NewTestTx(t)
	_, tx2 := kv.NewTestTx(t)

	generateBlocks(t, 1, 50, plainWriterGen(tx1), staticCodeStaticIncarnations)
	generateBlocks(t, 1, 100, plainWriterGen(tx2), staticCodeStaticIncarnations)

	err := stages.SaveStageProgress(tx2, stages.Execution, 100)
	assert.NoError(err)

	u := &UnwindState{ID: stages.Execution, UnwindPoint: 50}
	s := &StageState{ID: stages.Execution, BlockNumber: 100}
	err = UnwindExecutionStage(u, s, tx2, ctx, ExecuteBlockCfg{}, false)
	assert.NoError(err)

	compareCurrentState(t, tx1, tx2, dbutils.PlainStateBucket, dbutils.PlainContractCodeBucket, dbutils.ContractTEVMCodeBucket)
}

func TestUnwindExecutionStagePlainWithIncarnationChanges(t *testing.T) {
	ctx, assert := context.Background(), assert.New(t)
	_, tx1 := kv.NewTestTx(t)
	_, tx2 := kv.NewTestTx(t)

	generateBlocks(t, 1, 50, plainWriterGen(tx1), changeCodeWithIncarnations)
	generateBlocks(t, 1, 100, plainWriterGen(tx2), changeCodeWithIncarnations)

	err := stages.SaveStageProgress(tx2, stages.Execution, 100)
	assert.NoError(err)

	u := &UnwindState{ID: stages.Execution, UnwindPoint: 50}
	s := &StageState{ID: stages.Execution, BlockNumber: 100}
	err = UnwindExecutionStage(u, s, tx2, ctx, ExecuteBlockCfg{}, false)
	assert.NoError(err)

	compareCurrentState(t, tx1, tx2, dbutils.PlainStateBucket, dbutils.PlainContractCodeBucket)
}

func TestUnwindExecutionStagePlainWithCodeChanges(t *testing.T) {
	t.Skip("not supported yet, to be restored")
	ctx := context.Background()
	_, tx1 := kv.NewTestTx(t)
	_, tx2 := kv.NewTestTx(t)

	generateBlocks(t, 1, 50, plainWriterGen(tx1), changeCodeIndepenentlyOfIncarnations)
	generateBlocks(t, 1, 100, plainWriterGen(tx2), changeCodeIndepenentlyOfIncarnations)

	err := stages.SaveStageProgress(tx2, stages.Execution, 100)
	if err != nil {
		t.Errorf("error while saving progress: %v", err)
	}
	u := &UnwindState{ID: stages.Execution, UnwindPoint: 50}
	s := &StageState{ID: stages.Execution, BlockNumber: 100}
	err = UnwindExecutionStage(u, s, tx2, ctx, ExecuteBlockCfg{}, false)
	if err != nil {
		t.Errorf("error while unwinding state: %v", err)
	}

	compareCurrentState(t, tx1, tx2, dbutils.PlainStateBucket, dbutils.PlainContractCodeBucket)
}

func TestPruneExecution(t *testing.T) {
	ctx, assert := context.Background(), assert.New(t)
	_, tx := kv.NewTestTx(t)

	generateBlocks(t, 1, 50, plainWriterGen(tx), changeCodeIndepenentlyOfIncarnations)
	err := stages.SaveStageProgress(tx, stages.Execution, 50)
	assert.NoError(err)

	available, err := changeset.AvailableFrom(tx)
	assert.NoError(err)
	assert.Equal(uint64(1), available)

	s := &PruneState{ID: stages.Execution, ForwardProgress: 50}
	// check pruning distance > than current stage progress
	err = PruneExecutionStage(s, tx, ExecuteBlockCfg{prune: prune.Mode{History: 100, Receipts: 101, CallTraces: 200}}, ctx, false)
	assert.NoError(err)

	available, err = changeset.AvailableFrom(tx)
	assert.NoError(err)
	assert.Equal(uint64(1), available)
	available, err = changeset.AvailableStorageFrom(tx)
	assert.NoError(err)
	assert.Equal(uint64(1), available)

	// pruning distance, first run
	err = PruneExecutionStage(s, tx, ExecuteBlockCfg{prune: prune.Mode{History: 5, Receipts: 15, CallTraces: 25}}, ctx, false)
	assert.NoError(err)

	available, err = changeset.AvailableFrom(tx)
	assert.NoError(err)
	assert.Equal(uint64(45), available)
	available, err = changeset.AvailableStorageFrom(tx)
	assert.NoError(err)
	assert.Equal(uint64(45), available)

	// pruning distance, second run
	err = PruneExecutionStage(s, tx, ExecuteBlockCfg{prune: prune.Mode{History: 5, Receipts: 15, CallTraces: 25}}, ctx, false)
	assert.NoError(err)

	available, err = changeset.AvailableFrom(tx)
	assert.NoError(err)
	assert.Equal(uint64(45), available)
	available, err = changeset.AvailableStorageFrom(tx)
	assert.NoError(err)
	assert.Equal(uint64(45), available)
}