// Code generated by github.com/filecoin-project/venus/venus-devtool/api-gen. DO NOT EDIT.
package wallet

import (
	"context"

	address "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-state-types/crypto"

	"github.com/filecoin-project/venus/venus-shared/types"
)

type IWalletStruct struct {
	Internal struct {
		WalletDelete func(ctx context.Context, addr address.Address) error                                                           `perm:"admin"`
		WalletExport func(ctx context.Context, addr address.Address) (*types.KeyInfo, error)                                         `perm:"admin"`
		WalletHas    func(ctx context.Context, address address.Address) (bool, error)                                                `perm:"read"`
		WalletImport func(ctx context.Context, ki *types.KeyInfo) (address.Address, error)                                           `perm:"admin"`
		WalletList   func(ctx context.Context) ([]address.Address, error)                                                            `perm:"read"`
		WalletNew    func(ctx context.Context, kt types.KeyType) (address.Address, error)                                            `perm:"admin"`
		WalletSign   func(ctx context.Context, signer address.Address, toSign []byte, meta types.MsgMeta) (*crypto.Signature, error) `perm:"sign"`
	}
}

func (s *IWalletStruct) WalletDelete(p0 context.Context, p1 address.Address) error {
	return s.Internal.WalletDelete(p0, p1)
}
func (s *IWalletStruct) WalletExport(p0 context.Context, p1 address.Address) (*types.KeyInfo, error) {
	return s.Internal.WalletExport(p0, p1)
}
func (s *IWalletStruct) WalletHas(p0 context.Context, p1 address.Address) (bool, error) {
	return s.Internal.WalletHas(p0, p1)
}
func (s *IWalletStruct) WalletImport(p0 context.Context, p1 *types.KeyInfo) (address.Address, error) {
	return s.Internal.WalletImport(p0, p1)
}
func (s *IWalletStruct) WalletList(p0 context.Context) ([]address.Address, error) {
	return s.Internal.WalletList(p0)
}
func (s *IWalletStruct) WalletNew(p0 context.Context, p1 types.KeyType) (address.Address, error) {
	return s.Internal.WalletNew(p0, p1)
}
func (s *IWalletStruct) WalletSign(p0 context.Context, p1 address.Address, p2 []byte, p3 types.MsgMeta) (*crypto.Signature, error) {
	return s.Internal.WalletSign(p0, p1, p2, p3)
}

type IWalletLockStruct struct {
	Internal struct {
		Lock           func(ctx context.Context, password string) error `perm:"admin"`
		LockState      func(ctx context.Context) bool                   `perm:"admin"`
		SetPassword    func(ctx context.Context, password string) error `perm:"admin"`
		Unlock         func(ctx context.Context, password string) error `perm:"admin"`
		VerifyPassword func(ctx context.Context, password string) error `perm:"admin"`
	}
}

func (s *IWalletLockStruct) Lock(p0 context.Context, p1 string) error { return s.Internal.Lock(p0, p1) }
func (s *IWalletLockStruct) LockState(p0 context.Context) bool        { return s.Internal.LockState(p0) }
func (s *IWalletLockStruct) SetPassword(p0 context.Context, p1 string) error {
	return s.Internal.SetPassword(p0, p1)
}
func (s *IWalletLockStruct) Unlock(p0 context.Context, p1 string) error {
	return s.Internal.Unlock(p0, p1)
}
func (s *IWalletLockStruct) VerifyPassword(p0 context.Context, p1 string) error {
	return s.Internal.VerifyPassword(p0, p1)
}

type ILocalWalletStruct struct {
	IWalletStruct
	IWalletLockStruct
}

type ICommonStruct struct {
	Internal struct {
		AuthNew     func(ctx context.Context, perms []auth.Permission) ([]byte, error) `perm:"admin"`
		AuthVerify  func(ctx context.Context, token string) ([]auth.Permission, error) `perm:"read"`
		LogList     func(context.Context) ([]string, error)                            `perm:"read"`
		LogSetLevel func(context.Context, string, string) error                        `perm:"write"`
		Version     func(ctx context.Context) (types.Version, error)                   `perm:"read"`
	}
}

func (s *ICommonStruct) AuthNew(p0 context.Context, p1 []auth.Permission) ([]byte, error) {
	return s.Internal.AuthNew(p0, p1)
}
func (s *ICommonStruct) AuthVerify(p0 context.Context, p1 string) ([]auth.Permission, error) {
	return s.Internal.AuthVerify(p0, p1)
}
func (s *ICommonStruct) LogList(p0 context.Context) ([]string, error) { return s.Internal.LogList(p0) }
func (s *ICommonStruct) LogSetLevel(p0 context.Context, p1 string, p2 string) error {
	return s.Internal.LogSetLevel(p0, p1, p2)
}
func (s *ICommonStruct) Version(p0 context.Context) (types.Version, error) {
	return s.Internal.Version(p0)
}

type IWalletEventStruct struct {
	Internal struct {
		AddNewAddress     func(ctx context.Context, newAddrs []address.Address) error `perm:"admin"`
		AddSupportAccount func(ctx context.Context, supportAccount string) error      `perm:"admin"`
	}
}

func (s *IWalletEventStruct) AddNewAddress(p0 context.Context, p1 []address.Address) error {
	return s.Internal.AddNewAddress(p0, p1)
}
func (s *IWalletEventStruct) AddSupportAccount(p0 context.Context, p1 string) error {
	return s.Internal.AddSupportAccount(p0, p1)
}

type IFullAPIStruct struct {
	ILocalWalletStruct
	ICommonStruct
	IWalletEventStruct
}
