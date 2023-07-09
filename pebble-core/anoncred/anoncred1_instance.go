package anoncred

import (
	"fmt"
	"os"
)

var AnonCred1Instance CredentialSystem

func init() {
	params, err := os.ReadFile("anoncred1-params.bin")
	if err != nil {
		return
	}
	credSys := new(AnonCred1)
	err = credSys.FromBytes(params)
	if err != nil {
		return
	}
	AnonCred1Instance = credSys
	fmt.Println("AnonCred1Instance initialized...")
}
