package client

import (
	"github.com/bsv-blockchain/go-sdk/chainhash"
)

type BlockHeader struct {
	Height        uint32         `json:"height"`
	Hash          chainhash.Hash `json:"hash"`
	Version       uint32         `json:"version"`
	MerkleRoot    chainhash.Hash `json:"merkleRoot"`
	Timestamp     uint32         `json:"creationTimestamp"`
	Bits          uint32         `json:"difficultyTarget"`
	Nonce         uint32         `json:"nonce"`
	PreviousBlock chainhash.Hash `json:"prevBlockHash"`
}

type BlockHeaderState struct {
	Header BlockHeader `json:"header"`
	State  string      `json:"state"`
	Height uint32      `json:"height"`
}
