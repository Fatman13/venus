package lookup

import (
	"context"

	"gx/ipfs/QmQsErDt8Qgw1XrsXf2BpEzDgGWtB1YLsTAARBup5b6B9W/go-libp2p-peer"
	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/core"
	"github.com/filecoin-project/go-filecoin/vm"
)

// PeerLookupService provides an interface through which callers look up a miner's libp2p identity by their Filecoin address.
type PeerLookupService interface {
	GetPeerIDByMinerAddress(context.Context, address.Address) (peer.ID, error)
}

// ChainLookupService is a ChainManager-backed implementation of the PeerLookupService interface.
type ChainLookupService struct {
	chainManager           *core.ChainManager
	queryMethodFromAddress func() (address.Address, error)
}

var _ PeerLookupService = &ChainLookupService{}

// NewChainLookupService creates a new ChainLookupService from a ChainManager and a Wallet.
func NewChainLookupService(manager *core.ChainManager, queryMethodFromAddr func() (address.Address, error)) *ChainLookupService {
	return &ChainLookupService{
		chainManager:           manager,
		queryMethodFromAddress: queryMethodFromAddr,
	}
}

// GetPeerIDByMinerAddress attempts to get a miner's libp2p identity by loading the actor from the state tree and sending
// it a "getPeerID" message. The MinerActor is currently the only type of actor which has a peer ID.
func (c *ChainLookupService) GetPeerIDByMinerAddress(ctx context.Context, minerAddr address.Address) (peer.ID, error) {
	st, err := c.chainManager.State(ctx, c.chainManager.GetHeaviestTipSet().ToSlice())
	if err != nil {
		return peer.ID(""), errors.Wrap(err, "failed to load state tree")
	}
	addr, err := c.queryMethodFromAddress()
	if err != nil {
		return peer.ID(""), errors.Wrap(err, "failed to obtain a default from-address")
	}

	vms := vm.NewStorageMap(c.chainManager.Blockstore)
	retValue, retCode, err := core.CallQueryMethod(ctx, st, vms, minerAddr, "getPeerID", []byte{}, addr, nil)
	if err != nil {
		return peer.ID(""), errors.Wrapf(err, "failed to query local state tree(from %s, miner %s)", addr.String(), minerAddr.String())
	}

	if retCode != 0 {
		return peer.ID(""), errors.Wrap(err, "non-zero status code from getPeerID")
	}

	pid, err := peer.IDFromBytes(retValue[0])
	if err != nil {
		return peer.ID(""), errors.Wrap(err, "could not decode to peer.ID from message-bytes")
	}

	return pid, nil
}
