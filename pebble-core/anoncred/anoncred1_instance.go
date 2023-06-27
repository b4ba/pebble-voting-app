package anoncred

import (
	"fmt"
	"os"
)

var AnonCred1Instance CredentialSystem

/*
Go automatically calls the init() function during the program's initialization phase.

The init() function is used for initialization tasks that need to be performed before the program starts execution.
It is commonly used to set up variables, perform initialization logic, register components, or execute any other necessary setup operations.
*/

func init() {
	params, err := os.ReadFile("anoncred1-params.bin")
	if err != nil {
		return
	}
	credSys := new(AnonCred1)
	fmt.Println("credSys initialized: ", credSys)
	err = credSys.FromBytes(params)
	if err != nil {
		return
	}
	AnonCred1Instance = credSys
	fmt.Println("AnonCred1Instance initialized")
	fmt.Println()
	fmt.Println("AnonCred1Instance: ", AnonCred1Instance)
}
