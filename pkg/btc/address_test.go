package btc

import (
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"

	"github.com/btcsuite/btcd/btcec"
	"github.com/keep-network/keep-tecdsa/pkg/ecdsa"
)

func TestPublicKeyToWitnessPubKeyHashAddress(t *testing.T) {
	// Test data from [BIP-173] examples.
	publicKeyBytes, _ := hex.DecodeString("0279BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798")
	expectedAddress := "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx"

	btcecPublicKey, err := btcec.ParsePubKey(publicKeyBytes, btcec.S256())
	if err != nil {
		t.Fatal(err)
	}

	address, err := PublicKeyToWitnessPubKeyHashAddress(
		(*ecdsa.PublicKey)(btcecPublicKey),
		&chaincfg.TestNet3Params,
	)
	if err != nil {
		t.Errorf("unexpected error: [%v]", err)
	}

	if address != expectedAddress {
		t.Errorf(
			"unexpected address\nexpected: %v\nactual:   %v\n",
			expectedAddress,
			address,
		)
	}
}