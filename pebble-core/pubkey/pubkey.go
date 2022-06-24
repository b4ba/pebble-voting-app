package pubkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"

	"blockwatch.cc/tzgo/tezos"
	"github.com/giry-dev/pebble-voting-app/pebble-core/util"
)

type PublicKey []byte

type PrivateKey struct {
	t KeyType
	p PublicKey
	s []byte
}

type KeyType byte

const (
	KeyTypeEd25519 KeyType = iota
	KeyTypeTezos
)

var (
	ErrInvalidKeyLength = errors.New("pebble: invalid key length")

	ErrUnknownKeyType = errors.New("pebble: unknown key type")

	ErrInvalidSignature = errors.New("pebble: invalid signature")
)

func (k PrivateKey) Type() KeyType {
	return k.t
}

func (k PrivateKey) Public() PublicKey {
	return k.p
}

func (k PrivateKey) Secret() []byte {
	return k.s
}

func GenerateKey(keyType KeyType) (k PrivateKey, err error) {
	switch keyType {
	case KeyTypeEd25519:
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return k, err
		}
		return PrivateKey{keyType, PublicKey(pub), priv.Seed()}, nil
	case KeyTypeTezos:
		priv, err := tezos.GenerateKey(tezos.KeyTypeEd25519)
		if err != nil {
			return k, err
		}
		pub := priv.Public()
		return PrivateKey{keyType, PublicKey(pub.Bytes()), []byte(priv.String())}, nil
	default:
		return k, ErrUnknownKeyType
	}
}

func (k PrivateKey) Sign(msg []byte) ([]byte, error) {
	switch k.t {
	case KeyTypeEd25519:
		return ed25519.NewKeyFromSeed(k.s).Sign(rand.Reader, msg, nil)
	case KeyTypeTezos:
		key, err := tezos.ParsePrivateKey(string(k.s))
		if err != nil {
			return nil, err
		}
		hash := util.Hash(msg)
		sig, err := key.Sign(hash[:])
		if err != nil {
			return nil, err
		}
		return sig.Bytes(), nil
	default:
		return nil, ErrUnknownKeyType
	}
}

func (k PublicKey) Verify(msg, sig []byte) error {
	if len(k) == 0 {
		return ErrInvalidKeyLength
	}
	switch k[0] {
	case byte(KeyTypeEd25519):
		pk := ed25519.PublicKey(k[1:])
		if len(pk) != ed25519.PublicKeySize {
			return ErrInvalidKeyLength
		}
		if !ed25519.Verify(pk, msg, sig) {
			return ErrInvalidSignature
		}
		return nil
	case byte(KeyTypeTezos):
		pk, err := tezos.DecodeKey(k[1:])
		if err != nil {
			return err
		}
		var tzsig tezos.Signature
		err = tzsig.UnmarshalBinary(sig)
		if err != nil {
			return ErrInvalidSignature
		}
		hash := util.Hash(msg)
		err = pk.Verify(hash[:], tzsig)
		if err != nil {
			return ErrInvalidSignature
		}
		return nil
	default:
		return ErrUnknownKeyType
	}
}
