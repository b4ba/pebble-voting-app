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

type Invitation struct {
	Network string
	Address []byte
	Servers []string
}

// func (inv Invitation) String() string {
// 	var w util.BufferWriter
// 	w.WriteUint32(invitationVersion)
// 	w.WriteVector(inv.Address)
// 	w.WriteByte(byte(len(inv.Servers)))
// 	for _, s := range inv.Servers {
// 		w.WriteVector([]byte(s))
// 	}
// 	return base32c.CheckEncode(w.Buffer)
// }

// Includes Network in Invitation
func (inv Invitation) String() string {
	var w util.BufferWriter
	w.WriteUint32(invitationVersion)
	w.WriteVector([]byte(inv.Network)) // Added this line
	w.WriteVector(inv.Address)
	w.WriteByte(byte(len(inv.Servers)))
	for _, s := range inv.Servers {
		w.WriteVector([]byte(s))
	}
	return base32c.CheckEncode(w.Buffer)
}

// func DecodeInvitation(s string) (inv Invitation, err error) {
// 	p, err := base32c.CheckDecode(s)
// 	if err != nil {
// 		return inv, err
// 	}
// 	r := util.NewBufferReader(p)
// 	v, err := r.ReadUint32()
// 	if err != nil {
// 		return
// 	}
// 	if v != invitationVersion {
// 		return inv, ErrUnknownInvitationVersion
// 	}
// 	inv.Address, err = r.ReadVector()
// 	if err != nil {
// 		return
// 	}
// 	numServers, err := r.ReadByte()
// 	if err != nil {
// 		return
// 	}
// 	inv.Servers = make([]string, numServers)
// 	for i := range inv.Servers {
// 		fmt.Println("Reading server...")
// 		fmt.Println(i)
// 		b, err := r.ReadVector()
// 		if err != nil {
// 			return inv, err
// 		}
// 		inv.Servers[i] = string(b)
// 		fmt.Println(inv.Servers[i])
// 	}
// 	return
// }

// Includes the network in the invitation
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
	// Added these lines
	b, err := r.ReadVector()
	if err != nil {
		return
	}
	inv.Network = string(b)

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
