/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wallet

import (
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/hyperledger/aries-framework-go/pkg/wallet"
)

// CloseableWallet mock wallet
type CloseableWallet struct {
	CreateEncryptionKeyValue string
	CreateEncryptionKeyErr   error
	CreateSigningKeyValue    string
	CreateSigningKeyErr      error
	SignMessageValue         []byte
	SignMessageErr           error
	PackValue                []byte
	PackErr                  error
	UnpackValue              *wallet.Envelope
	UnpackErr                error
	MockDID                  *did.Doc
}

// Close previously-opened wallet, removing it if so configured.
func (m *CloseableWallet) Close() error {
	return nil
}

// CreateEncryptionKey create a new public/private encryption keypair.
func (m *CloseableWallet) CreateEncryptionKey() (string, error) {
	return m.CreateEncryptionKeyValue, m.CreateEncryptionKeyErr
}

// CreateSigningKey create a new public/private signing keypair.
func (m *CloseableWallet) CreateSigningKey() (string, error) {
	return m.CreateSigningKeyValue, m.CreateSigningKeyErr
}

// SignMessage sign a message using the private key associated with a given verification key.
func (m *CloseableWallet) SignMessage(message []byte, fromVerKey string) ([]byte, error) {
	return m.SignMessageValue, m.SignMessageErr
}

// DecryptMessage decrypt message
func (m *CloseableWallet) DecryptMessage(encMessage []byte, toVerKey string) ([]byte, string, error) {
	return nil, "", nil
}

// PackMessage Pack a message for one or more recipients.
func (m *CloseableWallet) PackMessage(envelope *wallet.Envelope) ([]byte, error) {
	return m.PackValue, m.PackErr
}

// UnpackMessage Unpack a message.
func (m *CloseableWallet) UnpackMessage(encMessage []byte) (*wallet.Envelope, error) {
	return m.UnpackValue, m.UnpackErr
}

// CreateDID returns new DID Document
func (m *CloseableWallet) CreateDID(method string, opts ...wallet.DocOpts) (*did.Doc, error) {
	return m.MockDID, nil
}
