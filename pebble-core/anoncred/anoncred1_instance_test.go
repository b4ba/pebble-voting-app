package anoncred

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"
)

var AnonCred1InstanceTest CredentialSystem

func TestInit(t *testing.T) {
	params, err := os.ReadFile("anoncred1-params.bin")
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	hexStr := hex.EncodeToString(params)
	fmt.Println("Hexadecimal representation:", hexStr)

	decodedBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		fmt.Println("Error decoding hex string:", err)
		return
	}

	fmt.Println("Decoded bytes:", decodedBytes)
	// fmt.Println("Decoded string:", string(decodedBytes))

	credSys := new(AnonCred1)
	err = credSys.FromBytes(params)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	fmt.Println("credSys", credSys)
	AnonCred1InstanceTest = credSys

}
