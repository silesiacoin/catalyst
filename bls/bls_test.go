package bls_test

import (
	"fmt"
	"github.com/ethereum/go-ethereum/bls/herumi"
	"github.com/ethereum/go-ethereum/bytesutil"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	assert "github.com/ethereum/go-ethereum/testutil/assert"
	"github.com/ethereum/go-ethereum/testutil/require"
	"math/big"
	"testing"
)

var (
	testKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddr    = crypto.PubkeyToAddress(testKey.PublicKey)
	testBalance = big.NewInt(2e10)
)

func TestExtraDataBLS(t *testing.T) {
	genesis, blocks := generateTestChainWithBLSExtraData()
	n, err := node.New(&node.Config{})
	if err != nil {
		t.Fatalf("could not get node: %v", err)
	}
	ethservice, err := eth.New(n, &eth.Config{Genesis: genesis, Ethash: ethash.Config{PowMode: ethash.ModeFake}})
	if err != nil {
		t.Fatalf("can't create new ethereum service: %v", err)
	}
	if err := n.Start(); err != nil {
		t.Fatalf("can't start test node: %v", err)
	}
	if _, err := ethservice.BlockChain().InsertChain(blocks[1:9]); err != nil {
		t.Fatalf("can't import test blocks: %v", err)
	}
	ethservice.SetEtherbase(testAddr)
	api := eth.NewEth2API(ethservice)
	// Put the 10th block's tx in the pool and produce a new block
	api.AddBlockTxs(blocks[9])
	blockParams := eth.ProduceBlockParams{
		ParentHash: blocks[9].ParentHash(),
		Slot:       blocks[9].NumberU64(),
		Timestamp:  blocks[9].Time(),
	}
	execData, err := api.ProduceBlock(blockParams)
	if err != nil {
		t.Fatalf("error producing block, err=%v", err)
	}
	if len(execData.Transactions) != blocks[9].Transactions().Len() {
		t.Fatalf("invalid number of transactions %d != 1", len(execData.Transactions))
	}
	extraData := ethservice.BlockChain().CurrentBlock().Extra()
	assert.NotNil(t, extraData)
	b32 := bytesutil.ToBytes32(extraData)
	pk, err := herumi.SecretKeyFromBytes(b32[:])
	require.NoError(t, err)
	pk2, err := herumi.SecretKeyFromBytes(b32[:])
	require.NoError(t, err)
	assert.DeepEqual(t, pk.Marshal(), pk2.Marshal())
}

func generateTestChainWithBLSExtraData() (*core.Genesis, []*types.Block) {
	db := rawdb.NewMemoryDatabase()
	config := params.AllEthashProtocolChanges
	genesis := &core.Genesis{
		Config:    config,
		Alloc:     core.GenesisAlloc{testAddr: {Balance: testBalance}},
		Timestamp: 9000,
	}
	priv, err := herumi.RandKey()
	if err != nil {
		panic(fmt.Sprintf("can't create new ethereum service: %v", err))
	}
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra(priv.Marshal())
	}
	gblock := genesis.ToBlock(db)
	engine := ethash.NewFaker()
	blocks, _ := core.GenerateChain(config, gblock, engine, db, 10, generate)
	blocks = append([]*types.Block{gblock}, blocks...)
	return genesis, blocks
}
