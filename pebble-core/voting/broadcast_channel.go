package voting

import (
	"context"
	"errors"

	"github.com/giry-dev/pebble-voting-app/pebble-core/voting/structs"
)

// Represents a message that can be sent over the broadcast channel.
type Message struct {
	ElectionParams *ElectionParams
	Credential     *structs.CredentialMessage
	SignedBallot   *structs.SignedBallot
	Decryption     *structs.DecryptionMessage
}

/*
Specifies the methods that a broadcast channel implementation must provide.

Id(): returns the identifier of the election associated with the broadcast channel.
Params(ctx context.Context): retrieves the election parameters from the broadcast channel.
Get(ctx context.Context): retrieves the messages from the broadcast channel.
Post(ctx context.Context, m Message): posts a new message to the broadcast channel.
*/
type BroadcastChannel interface {
	Id() ElectionID
	Params(ctx context.Context) (*ElectionParams, error)
	Get(ctx context.Context) ([]Message, error)
	Post(ctx context.Context, m Message) error
}

var (
	ErrInvalidMessageType = errors.New("pebble: invalid message type")
	ErrInvalidMessageSize = errors.New("pebble: invalid message size")
)

/*
Serializes the Message struct into a byte slice.
Determines the message type based on which field is non-nil.
Calls the corresponding Bytes() method on the non-nil field to serialize its content.
Prepends the byte value representing the message type to the serialized payload.
*/
func (m Message) Bytes() []byte {
	var phase ElectionPhase
	var p []byte
	if m.ElectionParams != nil {
		phase = Setup
		p = m.ElectionParams.Bytes()
	} else if m.Credential != nil {
		phase = CredGen
		p = m.Credential.Bytes()
	} else if m.SignedBallot != nil {
		phase = Cast
		p = m.SignedBallot.Bytes()
	} else if m.Decryption != nil {
		phase = Tally
		p = m.Decryption.Bytes()
	} else {
		panic("pebble: invalid message type")
	}
	r := make([]byte, 1, len(p)+1)
	r[0] = byte(phase)
	r = append(r, p...)
	return r
}

/*
Deserializes a byte slice into a Message struct.
Extracts the message type byte from the first element of the byte slice.
Based on the message type, initializes the corresponding field of the Message struct and deserializes the remaining bytes using the respective FromBytes() method.
*/
func MessageFromBytes(p []byte) (m Message, err error) {
	if len(p) < 1 {
		return m, ErrInvalidMessageSize
	}
	switch ElectionPhase(p[0]) {
	case Setup:
		m.ElectionParams = new(ElectionParams)
		err = m.ElectionParams.FromBytes(p[1:])
	case CredGen:
		m.Credential = new(structs.CredentialMessage)
		err = m.Credential.FromBytes(p[1:])
	case Cast:
		m.SignedBallot = new(structs.SignedBallot)
		err = m.SignedBallot.FromBytes(p[1:])
	case Tally:
		m.Decryption = new(structs.DecryptionMessage)
		err = m.Decryption.FromBytes(p[1:])
	default:
		return m, ErrInvalidMessageType
	}
	return
}

type MockBroadcastChannel struct {
	messages []Message
	params   *ElectionParams
	id       ElectionID
}

func NewMockBroadcastChannel(id ElectionID, params *ElectionParams) *MockBroadcastChannel {
	return &MockBroadcastChannel{
		params: params,
		id:     id,
	}
}

func (bc *MockBroadcastChannel) Id() ElectionID {
	return bc.id
}

func (bc *MockBroadcastChannel) Params(ctx context.Context) (*ElectionParams, error) {
	return bc.params, nil
}

func (bc *MockBroadcastChannel) Get(ctx context.Context) ([]Message, error) {
	return bc.messages, nil
}

func (bc *MockBroadcastChannel) Post(ctx context.Context, m Message) error {
	bc.messages = append(bc.messages, m)
	return nil
}
