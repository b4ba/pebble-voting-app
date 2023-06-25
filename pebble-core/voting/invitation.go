package voting

import (
	"errors"

	"github.com/giry-dev/pebble-voting-app/pebble-core/base32c"
	"github.com/giry-dev/pebble-voting-app/pebble-core/util"
)

var (
	ErrUnknownInvitationVersion = errors.New("pebble: unknown invitation version")
	ErrInvalidInvitation        = errors.New("pebble: invalid invitation")
)

const invitationVersion uint32 = 0x1b68c700

// Represents an invitation to join a network or participate in an activity.
// Contains the network name, address, and a list of servers associated with the invitation.
type Invitation struct {
	Network string
	Address []byte
	Servers []string
}

/*
Converts the Invitation struct into a string representation.
Serializes the invitation data by encoding it with base32c encoding.
Returns the encoded string.
*/
func (inv Invitation) String() string {
	var w util.BufferWriter
	w.WriteUint32(invitationVersion)
	w.WriteVector(inv.Address)
	w.WriteByte(byte(len(inv.Servers)))
	for _, s := range inv.Servers {
		w.WriteVector([]byte(s))
	}
	return base32c.CheckEncode(w.Buffer)
}

/*
Decodes the encoded invitation string and returns the corresponding Invitation struct.
Takes the encoded invitation string as input.
Decodes the base32c-encoded string to obtain the byte slice.
Reads the version from the byte slice and verifies that it matches the expected invitation version.
Reads the address and server information from the byte slice.
Constructs and returns the Invitation struct with the decoded data.
*/
func DecodeInvitation(s string) (inv Invitation, err error) {
	p, err := base32c.CheckDecode(s)
	if err != nil {
		return inv, err
	}
	r := util.NewBufferReader(p)
	v, err := r.ReadUint32()
	if err != nil {
		return
	}
	if v != invitationVersion {
		return inv, ErrUnknownInvitationVersion
	}
	inv.Address, err = r.ReadVector()
	if err != nil {
		return
	}
	numServers, err := r.ReadByte()
	if err != nil {
		return
	}
	inv.Servers = make([]string, numServers)
	for i := range inv.Servers {
		b, err := r.ReadVector()
		if err != nil {
			return inv, err
		}
		inv.Servers[i] = string(b)
	}
	return
}
