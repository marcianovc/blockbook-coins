package pirate

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/marcianovc/blockbook-coins/bchain"
	"github.com/marcianovc/blockbook-coins/bchain/coins/btc"
	"github.com/juju/errors"
)

type PirateRPC struct {
	*btc.BitcoinRPC
}

func NewPirateRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}
	s := &PirateRPC{
		b.(*btc.BitcoinRPC),
	}
	s.RPCMarshaler = btc.JSONMarshalerV1{}
	s.ChainConfig.SupportsEstimateSmartFee = false
	return s, nil
}

func (b *PirateRPC) Initialize() error {
	ci, err := b.GetChainInfo()
	if err != nil {
		return err
	}
	chainName := ci.Chain
	params := GetChainParams(chainName)
	b.Parser = NewPirateParser(params, b.ChainConfig)
	b.Testnet = false
	b.Network = "livenet"
	glog.Info("rpc: block chain ", params.Name)
	return nil
}

func (b *PirateRPC) GetBlock(hash string, height uint32) (*bchain.Block, error) {
	var err error
	if hash == "" && height > 0 {
		hash, err = b.GetBlockHash(height)
		if err != nil {
			return nil, err
		}
	}
	glog.V(1).Info("rpc: getblock (verbosity=1) ", hash)
	res := btc.ResGetBlockThin{}
	req := btc.CmdGetBlock{Method: "getblock"}
	req.Params.BlockHash = hash
	req.Params.Verbosity = 1
	err = b.Call(&req, &res)
	if err != nil {
		return nil, errors.Annotatef(err, "hash %v", hash)
	}
	if res.Error != nil {
		return nil, errors.Annotatef(res.Error, "hash %v", hash)
	}
	txs := make([]bchain.Tx, 0, len(res.Result.Txids))
	for _, txid := range res.Result.Txids {
		tx, err := b.GetTransaction(txid)
		if err != nil {
			if err == bchain.ErrTxNotFound {
				glog.Errorf("rpc: getblock: skipping transanction in block %s due error: %s", hash, err)
				continue
			}
			return nil, err
		}
		txs = append(txs, *tx)
	}
	block := &bchain.Block{
		BlockHeader: res.Result.BlockHeader,
		Txs:         txs,
	}
	return block, nil
}

func (b *PirateRPC) GetTransactionForMempool(txid string) (*bchain.Tx, error) {
	return b.GetTransaction(txid)
}


type cmdGetInfo struct {
	Method string `json:"method"`
}

type resGetInfo struct {
	Error  *bchain.RPCError `json:"error"`
	Result struct {
		Chain         string `json:"chain"`
		Blocks        int    `json:"blocks"`
		Headers       int    `json:"headers"`
		Bestblockhash string `json:"bestblockhash"`
	} `json:"result"`
}

func (b *PirateRPC) GetBlockChainInfo() (string, error) {
	glog.V(1).Info("rpc: getinfo")

	res := resGetInfo{}
	req := cmdGetInfo{Method: "getinfo"}
	err := b.Call(&req, &res)
	if err != nil {
		return "", err
	}
	if res.Error != nil {
		return "", res.Error
	}
	return res.Result.Chain, nil
}