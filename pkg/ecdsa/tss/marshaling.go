package tss

import (
	"fmt"
	"math/big"

	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/paillier"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/keep-network/keep-tecdsa/pkg/ecdsa/tss/gen/pb"
)

// Marshal converts ThresholdSigner to byte array.
func (s *ThresholdSigner) Marshal() ([]byte, error) {
	// Threshold key
	keygenData, err := s.thresholdKey.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to convert local party save data: [%v]", err)
	}

	// Group Info
	groupMemberIDs := make([][]byte, len(s.groupMemberIDs))
	for i, memberID := range s.groupMemberIDs {
		groupMemberIDs[i] = memberID
	}

	group := &pb.ThresholdSigner_GroupInfo{
		GroupID:            s.groupID,
		MemberID:           s.memberID,
		GroupMemberIDs:     groupMemberIDs,
		DishonestThreshold: int32(s.dishonestThreshold),
	}

	return (&pb.ThresholdSigner{
		GroupInfo:    group,
		ThresholdKey: keygenData,
	}).Marshal()
}

// Unmarshal converts a byte array back to ThresholdSigner.
func (s *ThresholdSigner) Unmarshal(bytes []byte) error {
	pbSigner := pb.ThresholdSigner{
		GroupInfo: &pb.ThresholdSigner_GroupInfo{},
	}
	if err := pbSigner.Unmarshal(bytes); err != nil {
		return fmt.Errorf("failed to unmarshal signer: [%v]", err)
	}

	// Threshold key
	s.thresholdKey = ThresholdKey{}
	if err := s.thresholdKey.Unmarshal(pbSigner.GetThresholdKey()); err != nil {
		return fmt.Errorf("failed to unmarshal signer: [%v]", err)
	}

	// Group Info
	pbGroupInfo := pbSigner.GetGroupInfo()

	groupMemberIDs := make([]MemberID, len(pbGroupInfo.GetGroupMemberIDs()))
	for i, memberID := range pbGroupInfo.GetGroupMemberIDs() {
		groupMemberIDs[i] = MemberID(memberID)
	}

	s.groupInfo = &groupInfo{
		groupID:            pbGroupInfo.GetGroupID(),
		memberID:           MemberID(pbGroupInfo.GetMemberID()),
		groupMemberIDs:     groupMemberIDs,
		dishonestThreshold: int(pbGroupInfo.GetDishonestThreshold()),
	}

	return nil
}

// Marshal converts thresholdKey to byte array.
func (tk *ThresholdKey) Marshal() ([]byte, error) {
	localPreParams := &pb.LocalPartySaveData_LocalPreParams{
		PaillierSK: &pb.LocalPartySaveData_LocalPreParams_PrivateKey{
			PublicKey: tk.LocalPreParams.PaillierSK.PublicKey.N.Bytes(),
			LambdaN:   tk.LocalPreParams.PaillierSK.LambdaN.Bytes(),
			PhiN:      tk.LocalPreParams.PaillierSK.PhiN.Bytes(),
		},
		NTilde: tk.LocalPreParams.NTildei.Bytes(),
		H1I:    tk.LocalPreParams.H1i.Bytes(),
		H2I:    tk.LocalPreParams.H2i.Bytes(),
	}

	localSecrets := &pb.LocalPartySaveData_LocalSecrets{
		Xi:      tk.LocalSecrets.Xi.Bytes(),
		ShareID: tk.LocalSecrets.ShareID.Bytes(),
	}

	marshalBigIntSlice := func(bigInts []*big.Int) [][]byte {
		bytesSlice := make([][]byte, len(bigInts))
		for i, bigInt := range bigInts {
			bytesSlice[i] = bigInt.Bytes()
		}
		return bytesSlice
	}

	bigXj := make([][]byte, len(tk.BigXj))
	for i, bigX := range tk.BigXj {
		encoded, err := bigX.GobEncode()
		if err != nil {
			return nil, fmt.Errorf("failed to encode BigXj: [%v]", err)
		}
		bigXj[i] = encoded
	}

	paillierPKs := make([][]byte, len(tk.PaillierPKs))
	for i, paillierPK := range tk.PaillierPKs {
		paillierPKs[i] = paillierPK.N.Bytes()
	}

	ecdsaPub, err := tk.ECDSAPub.GobEncode()
	if err != nil {
		return nil, fmt.Errorf("failed to encode ECDSAPub: [%v]", err)
	}

	return (&pb.LocalPartySaveData{
		LocalPreParams: localPreParams,
		LocalSecrets:   localSecrets,
		Ks:             marshalBigIntSlice(tk.Ks),
		NTildej:        marshalBigIntSlice(tk.NTildej),
		H1J:            marshalBigIntSlice(tk.H1j),
		H2J:            marshalBigIntSlice(tk.H2j),
		BigXj:          bigXj,
		PaillierPKs:    paillierPKs,
		EcdsaPub:       ecdsaPub,
	}).Marshal()
}

// Unmarshal converts a byte array back to thresholdKey.
func (tk *ThresholdKey) Unmarshal(bytes []byte) error {
	pbData := pb.LocalPartySaveData{}
	if err := pbData.Unmarshal(bytes); err != nil {
		return fmt.Errorf("failed to unmarshal signer: [%v]", err)
	}

	paillierSK := &paillier.PrivateKey{
		PublicKey: paillier.PublicKey{
			N: new(big.Int).SetBytes(pbData.GetLocalPreParams().GetPaillierSK().GetPublicKey()),
		},
		LambdaN: new(big.Int).SetBytes(pbData.GetLocalPreParams().GetPaillierSK().GetLambdaN()),
		PhiN:    new(big.Int).SetBytes(pbData.GetLocalPreParams().GetPaillierSK().GetPhiN()),
	}

	tk.LocalPreParams = keygen.LocalPreParams{
		PaillierSK: paillierSK,
		NTildei:    new(big.Int).SetBytes(pbData.GetLocalPreParams().GetNTilde()),
		H1i:        new(big.Int).SetBytes(pbData.GetLocalPreParams().GetH1I()),
		H2i:        new(big.Int).SetBytes(pbData.GetLocalPreParams().GetH2I()),
	}

	tk.LocalSecrets = keygen.LocalSecrets{
		Xi:      new(big.Int).SetBytes(pbData.GetLocalSecrets().GetXi()),
		ShareID: new(big.Int).SetBytes(pbData.GetLocalSecrets().GetShareID()),
	}

	unmarshalBigIntSlice := func(bytesSlice [][]byte) []*big.Int {
		bigIntSlice := make([]*big.Int, len(bytesSlice))
		for i, bytes := range bytesSlice {
			bigIntSlice[i] = new(big.Int).SetBytes(bytes)
		}
		return bigIntSlice
	}

	tk.BigXj = make([]*crypto.ECPoint, len(pbData.GetBigXj()))
	for i, bigX := range pbData.GetBigXj() {
		tk.BigXj[i] = &crypto.ECPoint{}
		if err := tk.BigXj[i].GobDecode(bigX); err != nil {
			return fmt.Errorf("failed to decode BigXj: [%v]", err)
		}
	}

	tk.PaillierPKs = make([]*paillier.PublicKey, len(pbData.GetPaillierPKs()))
	for i, paillierPK := range pbData.GetPaillierPKs() {
		tk.PaillierPKs[i] = &paillier.PublicKey{
			N: new(big.Int).SetBytes(paillierPK),
		}
	}

	tk.ECDSAPub = &crypto.ECPoint{}
	if err := tk.ECDSAPub.GobDecode(pbData.GetEcdsaPub()); err != nil {
		return fmt.Errorf("failed to decode ECDSAPub: [%v]", err)
	}

	tk.Ks = unmarshalBigIntSlice(pbData.GetKs())
	tk.NTildej = unmarshalBigIntSlice(pbData.GetNTildej())
	tk.H1j = unmarshalBigIntSlice(pbData.GetH1J())
	tk.H2j = unmarshalBigIntSlice(pbData.GetH2J())

	return nil
}

// Marshal converts this message to a byte array suitable for network communication.
func (m *TSSProtocolMessage) Marshal() ([]byte, error) {
	return (&pb.TSSProtocolMessage{
		SenderID:    m.SenderID,
		Payload:     m.Payload,
		IsBroadcast: m.IsBroadcast,
	}).Marshal()
}

// Unmarshal converts a byte array produced by Marshal to a message.
func (m *TSSProtocolMessage) Unmarshal(bytes []byte) error {
	pbMsg := &pb.TSSProtocolMessage{}
	if err := pbMsg.Unmarshal(bytes); err != nil {
		return err
	}

	m.SenderID = MemberID(pbMsg.SenderID)
	m.Payload = pbMsg.Payload
	m.IsBroadcast = pbMsg.IsBroadcast

	return nil
}

// Marshal converts this message to a byte array suitable for network communication.
func (m *JoinMessage) Marshal() ([]byte, error) {
	return (&pb.JoinMessage{
		SenderID: m.SenderID,
	}).Marshal()
}

// Unmarshal converts a byte array produced by Marshal to a message.
func (m *JoinMessage) Unmarshal(bytes []byte) error {
	pbMsg := &pb.JoinMessage{}
	if err := pbMsg.Unmarshal(bytes); err != nil {
		return err
	}

	m.SenderID = MemberID(pbMsg.SenderID)

	return nil
}
