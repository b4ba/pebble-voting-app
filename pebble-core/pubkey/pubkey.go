package pubkey

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"strings"

	"blockwatch.cc/tzgo/tezos"
	"github.com/giry-dev/pebble-voting-app/pebble-core/base32c"
	"github.com/giry-dev/pebble-voting-app/pebble-core/util"
)

//  It represents a public key and is defined as a byte slice.
type PublicKey []byte

// It represents a private key and consists of a public key (PublicKey)
// and a byte slice representing the secret part of the key.
type PrivateKey struct {
	p PublicKey
	s []byte
}

type KeyType byte

const (
	KeyTypeUnknown KeyType = iota
	KeyTypeEd25519
	KeyTypeTezos
)

var (
	ErrInvalidKeyLength = errors.New("pebble: invalid key length")

	ErrUnknownKeyType = errors.New("pebble: unknown key type")

	ErrInvalidSignature = errors.New("pebble: invalid signature")
)

var noHashSignerOpts crypto.SignerOpts = crypto.Hash(0)

// Creates a new PublicKey by combining the key type and key data.
func newPublicKey(t KeyType, k []byte) PublicKey {
	p := make(PublicKey, len(k)+1)
	p[0] = byte(t)
	copy(p[1:], k)
	return p
}

// Returns the key type of a given PublicKey or PrivateKey instance.
func (k PublicKey) Type() KeyType {
	if len(k) < 1 {
		return KeyTypeUnknown
	}
	return KeyType(k[0])
}

func (k PrivateKey) Type() KeyType {
	return k.p.Type()
}

// Returns the public key part of a PrivateKey instance.
func (k PrivateKey) Public() PublicKey {
	return k.p
}

// Returns the secret key data of a PrivateKey instance.
func (k PrivateKey) Secret() []byte {
	return k.s
}

/*
It generates a new private key based on the specified key type.
The function supports key types KeyTypeEd25519 and KeyTypeTezos.
For KeyTypeEd25519, it uses the ed25519 package to generate the key pair.
For KeyTypeTezos, it uses the tezos package.
*/
func GenerateKey(keyType KeyType) (k PrivateKey, err error) {
	switch keyType {
	case KeyTypeEd25519:
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return k, err
		}
		return PrivateKey{newPublicKey(keyType, pub), priv.Seed()}, nil
	case KeyTypeTezos:
		priv, err := tezos.GenerateKey(tezos.KeyTypeEd25519)
		if err != nil {
			return k, err
		}
		pub := priv.Public()
		return PrivateKey{newPublicKey(keyType, pub.Bytes()), []byte(priv.String())}, nil
	default:
		return k, ErrUnknownKeyType
	}
}

// Signs a message using the private key.
func (k PrivateKey) Sign(msg []byte) ([]byte, error) {
	switch k.Type() {
	case KeyTypeEd25519:
		return ed25519.NewKeyFromSeed(k.s).Sign(rand.Reader, msg, noHashSignerOpts)
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

// Verifies the signature of a message using the corresponding public key.
func (k PublicKey) Verify(msg, sig []byte) error {
	if len(k) == 0 {
		return ErrInvalidKeyLength
	}
	switch KeyType(k[0]) {
	case KeyTypeEd25519:
		pk := ed25519.PublicKey(k[1:])
		if len(pk) != ed25519.PublicKeySize {
			return ErrInvalidKeyLength
		}
		if !ed25519.Verify(pk, msg, sig) {
			return ErrInvalidSignature
		}
		return nil
	case KeyTypeTezos:
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

/*
It converts a PublicKey to its string representation.
The function supports key types KeyTypeEd25519 and KeyTypeTezos.
For KeyTypeEd25519, it uses the base32c encoding.
For KeyTypeTezos, it converts the PublicKey to a tezos.Key type and returns its string representation.
*/
func (k PublicKey) String() (string, error) {
	if len(k) == 0 {
		return "", ErrInvalidKeyLength
	}
	switch KeyType(k[0]) {
	case KeyTypeEd25519:
		p := make([]byte, 2, len(k)+1)
		p[0] = 238
		p[1] = 78
		p = append(p, k[1:]...)
		return base32c.CheckEncode(p), nil
	case KeyTypeTezos:
		pk, err := tezos.DecodeKey(k[1:])
		if err != nil {
			return "", err
		}
		return pk.String(), nil
	default:
		return "", ErrUnknownKeyType
	}
}

// Parses a string representation of a public key and returns a PublicKey.
// The function supports parsing keys in both base32c and tz formats.
// func Parse(s string) (PublicKey, error) {
// 	if strings.HasPrefix(s, "EPK") {
// 		p, err := base32c.CheckDecode(s)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if len(p) < 3 || p[0] != 238 || p[1] != 78 {
// 			return nil, ErrUnknownKeyType
// 		}
// 		return PublicKey(p[2:]), nil
// 	} else if strings.HasPrefix(s, "tz") {
// 		var key tezos.Key
// 		err := key.UnmarshalText([]byte(s))
// 		if err != nil {
// 			return nil, err
// 		}
// 		keyBytes, err := key.MarshalBinary()
// 		if err != nil {
// 			return nil, err
// 		}
// 		pk := make(PublicKey, len(keyBytes)+1)
// 		pk[0] = byte(KeyTypeTezos)
// 		copy(pk[1:], keyBytes)
// 		return pk, nil
// 	}
// 	return nil, ErrUnknownKeyType
// }

// Updated version of Parse that also return the key type as prefix of the public key.
func Parse(s string) (PublicKey, error) {
	if strings.HasPrefix(s, "EPK") {
		p, err := base32c.CheckDecode(s)
		if err != nil {
			return nil, err
		}
		if len(p) < 3 || p[0] != 238 || p[1] != 78 {
			return nil, ErrUnknownKeyType
		}
		// Create a new public key with type and append the actual key part
		return newPublicKey(KeyTypeEd25519, p[2:]), nil //<-- here we add the key type
	} else if strings.HasPrefix(s, "tz") {
		var key tezos.Key
		err := key.UnmarshalText([]byte(s))
		if err != nil {
			return nil, err
		}
		keyBytes, err := key.MarshalBinary()
		if err != nil {
			return nil, err
		}
		// Create a new public key with type and append the actual key part
		return newPublicKey(KeyTypeTezos, keyBytes), nil
	}
	return nil, ErrUnknownKeyType
}

// Add a new function to validate Ed25519 public keys
func IsValidPublicKey(key string) (bool, error) {
	parsedKey, err := ParsePublicKey(key)
	if err != nil {
		return false, err
	}
	return parsedKey.Type() == KeyTypeEd25519, nil
}
