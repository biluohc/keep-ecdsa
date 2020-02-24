package tss

// TSSProtocolMessage is a network message used to transport messages generated in
// TSS protocol execution. It is a wrapper over a message generated by underlying
// implementation of the protocol.
type TSSProtocolMessage struct {
	SenderID    MemberID
	Payload     []byte
	IsBroadcast bool
}

// Type returns a string type of the `TSSMessage` so that it conforms to
// `net.Message` interface.
func (m *TSSProtocolMessage) Type() string {
	return "ecdsa/tss_message"
}

// JoinMessage is a network message used to notify peer members about readiness
// to start protocol execution.
type JoinMessage struct {
	SenderID MemberID
}

// Type returns a string type of the `JoinMessage`.
func (m *JoinMessage) Type() string {
	return "ecdsa/join_message"
}

// AnnounceMessage is a network message used to announce peer's presence.
type AnnounceMessage struct {
	SenderID MemberID
}

// Type returns a string type of the `AnnounceMessage`.
func (m *AnnounceMessage) Type() string {
	return "ecdsa/announce_message"
}
