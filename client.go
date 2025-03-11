package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bitcoin-sv/go-sdk/chainhash"
)

type HeadersClient struct {
	Ctx      context.Context
	Url      string
	ApiKey   string
	updated  time.Time
	chaintip *BlockHeader
	C        chan *BlockHeader
}

func (c *HeadersClient) IsValidRootForHeight(root *chainhash.Hash, height uint32) (bool, error) {
	if header, err := c.BlockByHeight(c.Ctx, height); err != nil {
		return false, err
	} else {
		return header.MerkleRoot.Equal(*root), nil
	}
}

func (c *HeadersClient) StartChaintipSub(ctx context.Context) {
	if c.C == nil {
		c.C = make(chan *BlockHeader, 1000)
	}
	go func() {
		for {
			if _, err := c.GetChaintip(ctx); err != nil {
				log.Panic(err)
			}
		}
	}()
}

func (c *HeadersClient) GetChaintip(ctx context.Context) (*BlockHeader, error) {
	if time.Since(c.updated) < 5*time.Second {
		return c.chaintip, nil
	}
	headerState := &BlockHeaderState{}
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/chain/tip/longest", c.Url), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	if res, err := client.Do(req); err != nil {
		return nil, err
	} else {
		defer res.Body.Close()
		if err := json.NewDecoder(res.Body).Decode(headerState); err != nil {
			return nil, err
		}
		header := &headerState.Header
		if c.C != nil && (c.chaintip == nil || header.Hash != c.chaintip.Hash) {
			c.C <- header
		}
		header.Height = headerState.Height
		c.chaintip = header
		c.updated = time.Now()
		return header, nil
	}
}

func (c *HeadersClient) BlockByHeight(ctx context.Context, height uint32) (*BlockHeader, error) {
	headers := []BlockHeader{}
	client := &http.Client{}
	url := fmt.Sprintf("%s/api/v1/chain/header/byHeight?height=%d", c.Url, height)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	if res, err := client.Do(req); err != nil {
		return nil, err
	} else {
		defer res.Body.Close()
		if err := json.NewDecoder(res.Body).Decode(&headers); err != nil {
			return nil, err
		}
		for _, header := range headers {
			if state, err := c.GetBlockState(ctx, header.Hash.String()); err != nil {
				return nil, err
			} else if state.State == "LONGEST_CHAIN" {
				header.Height = state.Height
				return &header, nil
			}
		}
		header := &headers[0]
		header.Height = height
		return header, nil
	}
}

func (c *HeadersClient) BlockByHash(ctx context.Context, hash string) (*BlockHeader, error) {
	if headerState, err := c.GetBlockState(ctx, hash); err != nil {
		return nil, err
	} else {
		header := &headerState.Header
		header.Height = headerState.Height
		return header, nil
	}
}

func (c *HeadersClient) GetBlockState(ctx context.Context, hash string) (*BlockHeaderState, error) {
	headerState := &BlockHeaderState{}
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/chain/header/state/%s", c.Url, hash), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	if res, err := client.Do(req); err != nil {
		return nil, err
	} else {
		defer res.Body.Close()
		if err := json.NewDecoder(res.Body).Decode(headerState); err != nil {
			return nil, err
		}
	}
	return headerState, nil
}

func (c *HeadersClient) Blocks(ctx context.Context, fromBlock uint32, count uint) ([]*BlockHeader, error) {
	headers := make([]*BlockHeader, 0, count)
	client := &http.Client{}
	url := fmt.Sprintf("%s/api/v1/chain/header/byHeight?height=%d&count=%d", c.Url, fromBlock, count)
	byHash := make(map[string]*BlockHeader)
	var results []*BlockHeader
	if req, err := http.NewRequest("GET", url, nil); err != nil {
		return nil, err
	} else {
		req.Header.Set("Authorization", "Bearer "+c.ApiKey)
		if res, err := client.Do(req); err != nil {
			return nil, err
		} else {
			defer res.Body.Close()
			if err := json.NewDecoder(res.Body).Decode(&headers); err != nil {
				return nil, err
			} else if len(headers) == 0 {
				return headers, nil
			}
			for _, header := range headers {
				byHash[header.Hash.String()] = header
			}
			for i := len(headers) - 1; i >= 0; i-- {
				lastHeader := headers[i]
				if state, err := c.GetBlockState(ctx, lastHeader.Hash.String()); err != nil {
					return nil, err
				} else if state.State == "LONGEST_CHAIN" {
					lastHeight := state.Height
					results = make([]*BlockHeader, lastHeight-fromBlock+1)
					block := &state.Header
					block.Height = state.Height
					results[block.Height-fromBlock] = block
					for {
						parent := block
						if block = byHash[parent.PreviousBlock.String()]; block != nil {
							block.Height = parent.Height - 1
							results[block.Height-fromBlock] = block
						} else {
							return results, nil
						}
					}
				}
			}

			return results, nil
		}
	}
}
