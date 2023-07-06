package pubkey

import (
	"fmt"
	"testing"
)

func TestGenerateKeyAndOperations(t *testing.T) {
	// Define the message and the KeyType
	message := []byte("Hello, World!")
	keyType := KeyTypeEd25519

	// Generate a key
	key, err := GenerateKey(keyType)
	if err != nil {
		fmt.Println("err: ", err)
	}
	fmt.Println("key: ", key)
	fmt.Println()
	fmt.Println("public key: ", key.Public())
	fmt.Println()
	fmt.Println("private key: ", key.Secret())

	// Check the key type
	if keyType == key.Type() {
		fmt.Println("keyType == key.Type()")
	}

	// Sign the message
	signature, err := key.Sign(message)
	if err != nil {
		fmt.Println("err: ", err)
	}
	fmt.Println("signature: ", signature)

	// Verify the signature
	err = key.Public().Verify(message, signature)
	if err != nil {
		fmt.Println("err: ", err)
	}

	// Check the string representation of the public key
	publicKeyStr, err := key.Public().String()
	if err != nil {
		fmt.Println("err: ", err)
	}
	fmt.Println("publicKeyStr: ", publicKeyStr)

	// Parse the public key from the string representation
	parsedKey, err := Parse(publicKeyStr)
	if err != nil {
		fmt.Println("err: ", err)
	}
	fmt.Println("parsedKey: ", parsedKey)

	// Check that the parsed key is the same as the original public key
	// require.Equal(t, key.Public(), parsedKey)
	if len(key.Public()) != len(parsedKey) {
		t.Errorf("Key and parsedKey lengths are not equal")
		return
	}

	for i := range key.Public() {
		if key.Public()[i] != parsedKey[i] {
			t.Errorf("Key and parsedKey elements at index %d are not equal", i)
			return
		}
	}
}
