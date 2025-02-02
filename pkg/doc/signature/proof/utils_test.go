/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package proof

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAddProof(t *testing.T) {
	doc := getDefaultDoc()
	proofs, err := GetProofs(doc)
	require.Equal(t, err, ErrProofNotFound)
	require.Nil(t, proofs)

	now := time.Now()
	proof1 := Proof{Creator: "creator-1",
		Created:    &now,
		ProofValue: []byte("proof"),
		Type:       "Ed25519Signature2018"}

	err = AddProof(doc, &proof1)
	require.NoError(t, err)

	proofs, err = GetProofs(doc)
	require.NoError(t, err)
	require.Equal(t, 1, len(proofs))

	proof2 := Proof{Creator: "creator-2",
		Created:    &now,
		ProofValue: []byte("proof"),
		Type:       "Ed25519Signature2018"}
	err = AddProof(doc, &proof2)
	require.NoError(t, err)

	proofs, err = GetProofs(doc)
	require.NoError(t, err)
	require.Equal(t, 2, len(proofs))
	require.Equal(t, "creator-1", proofs[0].Creator)
	require.Equal(t, "creator-2", proofs[1].Creator)
}

func TestGetCopyWithoutProof(t *testing.T) {
	doc := getDefaultDocWithSignature()
	proofs, err := GetProofs(doc)
	require.NoError(t, err)
	require.Equal(t, 1, len(proofs))

	docCopy := GetCopyWithoutProof(doc)

	proofs, err = GetProofs(docCopy)
	require.Equal(t, err, ErrProofNotFound)
	require.Nil(t, proofs)

	require.True(t, reflect.DeepEqual(docCopy, getDefaultDoc()))
}

func TestInvalidProofFormat(t *testing.T) {
	doc := map[string]interface{}{
		"test": "test",
		"proof": map[string]interface{}{
			"type":       "Ed25519Signature2018",
			"creator":    "creator",
			"created":    "2011-09-23T20:21:34Z",
			"proofValue": "ABC",
		},
	}
	proofs, err := GetProofs(doc)
	require.NotNil(t, err)
	require.Nil(t, proofs)
	require.Contains(t, err.Error(), "expecting []interface{}")

	now := time.Now()
	proof := Proof{Creator: "creator-2",
		Created:    &now,
		ProofValue: []byte("proof #2"),
		Type:       "Ed25519Signature2018"}

	err = AddProof(doc, &proof)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "expecting []interface{}")
}

func getDefaultDoc() map[string]interface{} {
	return map[string]interface{}{
		"test": "test",
	}
}

func getDefaultDocWithSignature() map[string]interface{} {
	return map[string]interface{}{
		"test":  "test",
		"proof": getProofs(),
	}
}

func getProofs() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"type":       "Ed25519Signature2018",
			"creator":    "creator",
			"created":    "2011-09-23T20:21:34Z",
			"proofValue": "ABC",
		},
	}
}
