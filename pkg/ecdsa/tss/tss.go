// Package tss contains implementation of Threshold Multi-Party ECDSA Signature
// Scheme. This package uses [tss-lib] protocol implementation based on [GG19].
//
// [tss-lib]: https://github.com/binance-chain/tss-lib.
// [GG19]: Fast Multiparty Threshold ECDSA with Fast Trustless Setup, Rosario
// Gennaro and Steven Goldfeder, 2019, https://eprint.iacr.org/2019/114.pdf.
package tss

import (
	"context"
	"fmt"
	"time"

	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/ipfs/go-log"
	"github.com/keep-network/keep-tecdsa/pkg/ecdsa"
	"github.com/keep-network/keep-tecdsa/pkg/net"
)

const (
	keyGenerationTimeout = 120 * time.Second
	signingTimeout       = 120 * time.Second
)

var logger = log.Logger("keep-tss")

// GenerateThresholdSigner executes a threshold multi-party key generation protocol.
//
// It expects unique identifiers of the current member as well as identifiers of
// all members of the signing group. Group ID should be unique for each concurrent
// execution.
//
// Dishonest threshold `t` defines a maximum number of signers controlled by the
// adversary such that the adversary still cannot produce a signature. Any subset
// of `t + 1` players can jointly sign, but any smaller subset cannot.
//
// TSS protocol requires pre-parameters such as safe primes to be generated for
// execution. The parameters should be generated prior to running this function.
// If not provided they will be generated.
//
// As a result a signer will be returned or an error, if key generation failed.
func GenerateThresholdSigner(
	groupID string,
	memberID MemberID,
	groupMemberIDs []MemberID,
	dishonestThreshold uint,
	networkProvider net.Provider,
	tssPreParams *keygen.LocalPreParams,
) (*ThresholdSigner, error) {
	if len(groupMemberIDs) < 2 {
		return nil, fmt.Errorf(
			"group should have at least 2 members but got: [%d]",
			len(groupMemberIDs),
		)
	}

	if len(groupMemberIDs) <= int(dishonestThreshold) {
		return nil, fmt.Errorf(
			"group size [%d], should be greater than dishonest threshold [%d]",
			len(groupMemberIDs),
			dishonestThreshold,
		)
	}

	group := &groupInfo{
		groupID:            groupID,
		memberID:           memberID,
		groupMemberIDs:     groupMemberIDs,
		dishonestThreshold: int(dishonestThreshold),
	}

	if tssPreParams == nil {
		// TODO: Should we return an error here? We expect the params to be provided
		// from pool but if they are not provided to this function they will
		// be generated by underlying tss-lib protocol implementation anyway.
		logger.Warningf(
			"tss pre-params were not provided, they will be generated on protocol execution",
		)
	}

	netBridge, err := newNetworkBridge(group, networkProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize network bridge: [%v]", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), keyGenerationTimeout)
	defer cancel()

	keyGenSigner, err := initializeKeyGeneration(
		ctx,
		group,
		tssPreParams,
		netBridge,
	)
	if err != nil {
		return nil, err
	}
	logger.Infof("[party:%s]: initialized key generation", keyGenSigner.keygenParty.PartyID())

	if err := joinProtocol(ctx, group, networkProvider); err != nil {
		return nil, fmt.Errorf("failed to join the protocol: [%v]", err)
	}

	logger.Infof("[party:%s]: starting key generation", keyGenSigner.keygenParty.PartyID())

	signer, err := keyGenSigner.generateKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: [%v]", err)
	}
	logger.Infof("[party:%s]: completed key generation", keyGenSigner.keygenParty.PartyID())

	return signer, nil
}

// CalculateSignature executes a threshold multi-party signature calculation
// protocol for the given digest. As a result the calculated ECDSA signature will
// be returned or an error, if the signature generation failed.
func (s *ThresholdSigner) CalculateSignature(
	digest []byte,
	networkProvider net.Provider,
) (*ecdsa.Signature, error) {
	netBridge, err := newNetworkBridge(s.groupInfo, networkProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize network bridge: [%v]", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), keyGenerationTimeout)
	defer cancel()

	signingSigner, err := s.initializeSigning(ctx, digest[:], netBridge)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize signing: [%v]", err)
	}

	if err := joinProtocol(ctx, s.groupInfo, networkProvider); err != nil {
		return nil, fmt.Errorf("failed to join the protocol:: [%v]", err)
	}

	signature, err := signingSigner.sign(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: [%v]", err)
	}

	return signature, err
}
