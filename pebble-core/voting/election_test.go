package voting

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/giry-dev/pebble-voting-app/pebble-core/anoncred"
	"github.com/giry-dev/pebble-voting-app/pebble-core/pubkey"
	"github.com/giry-dev/pebble-voting-app/pebble-core/util"
	"github.com/giry-dev/pebble-voting-app/pebble-core/vdf"
	"github.com/giry-dev/pebble-voting-app/pebble-core/voting/methods"
	"github.com/giry-dev/pebble-voting-app/pebble-core/voting/structs"
)

// This is a mock implementation of the SecretsManager interface.
// It stores the private key, secret credential, signed ballot, and VDF solution.
// It implements the methods required by the SecretsManager interface.
type mockSecretsManager struct {
	privateKey       pubkey.PrivateKey
	secretCredential anoncred.SecretCredential
	ballot           structs.SignedBallot
	solution         vdf.VdfSolution
}

func (sm *mockSecretsManager) GetPrivateKey() (pubkey.PrivateKey, error) {
	return sm.privateKey, nil
}

func (sm *mockSecretsManager) GetSecretCredential(sys anoncred.CredentialSystem) (anoncred.SecretCredential, error) {
	return sm.secretCredential, nil
}

func (sm *mockSecretsManager) GetBallot() (structs.SignedBallot, error) {
	return sm.ballot, nil
}

func (sm *mockSecretsManager) SetBallot(ballot structs.SignedBallot) error {
	sm.ballot = ballot
	return nil
}

func (sm *mockSecretsManager) GetVdfSolution() (vdf.VdfSolution, error) {
	return sm.solution, nil
}

func (sm *mockSecretsManager) SetVdfSolution(sol vdf.VdfSolution) error {
	sm.solution = sol
	return nil
}

// This function generates a specified number of secret credentials using the given credential system.
// It returns a slice of generated secret credentials.
func generateSecretCredentials(credSys anoncred.CredentialSystem, count int) (creds []anoncred.SecretCredential, err error) {
	creds = make([]anoncred.SecretCredential, count)
	for i := range creds {
		creds[i], err = credSys.GenerateSecretCredential()
		if err != nil {
			return nil, err
		}
	}
	return
}

// This function generates a specified number of private keys using the Ed25519 key type.
// It returns a slice of generated private keys.
func generatePrivateKeys(count int) (privs []pubkey.PrivateKey, err error) {
	privs = make([]pubkey.PrivateKey, count)
	for i := range privs {
		privs[i], err = pubkey.GenerateKey(pubkey.KeyTypeEd25519)
		if err != nil {
			return nil, err
		}
	}
	return
}

// This function generates an eligibility list based on the provided private keys.
// It creates an empty eligibility list and adds each private key's hash as a participant.
func generateEligibilityList(privs []pubkey.PrivateKey) (ell *structs.EligibilityList) {
	ell = structs.NewEligibilityList()
	for _, priv := range privs {
		ell.Add(util.Hash(priv.Public()), util.HashValue{})
	}
	return ell
}

// This function generates an ElectionParams struct using the provided eligibility list.
// It sets various parameters such as the start and end times, voting method, and choices.
// It returns the generated ElectionParams struct.
func generateElectionParams(ell *structs.EligibilityList) (params ElectionParams) {
	now := time.Now()
	params.EligibilityList = ell
	params.CastStart = now.Add(time.Second * 20)
	params.TallyStart = now.Add(time.Second * 40)
	params.TallyEnd = now.Add(time.Second * 60)
	params.VotingMethod = "Plurality"
	params.Choices = []string{"Toby Wilkinson", "Ava McLean", "Oliver Rogers"}
	return
}

/*
This is the main test function for the election.
It performs the following steps:
Sets up the credential system, private keys, eligibility list, and election parameters.
Creates a mock broadcast channel and secrets manager.
Initializes an Election instance with the necessary components.
Posts credentials for all private keys.
Waits until the cast phase starts.
Randomly selects a voter and casts a vote.
Waits until the tally phase starts.
Performs ballot decryption using the VDF solution.
Waits until the tally phase ends.
Retrieves the election progress.
*/
func TestElection(t *testing.T) {
	fmt.Println("Starting election test...")
	ctx := context.Background()
	fmt.Println("Generating credential system...")
	// It initializes the credential system (anoncred.AnonCred1)
	credSys := new(anoncred.AnonCred1)
	fmt.Println("SetupCircuit...")
	// and sets up the circuit with a difficulty of 8.
	err := credSys.SetupCircuit(8)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	fmt.Println("Generating private keys...")
	// Private keys are generated using the Ed25519 key type,
	privateKeys, err := generatePrivateKeys(10)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	fmt.Println("Generating eligibility list...")
	// and an eligibility list is created based on these private keys.
	elligibilityList := generateEligibilityList(privateKeys)
	fmt.Println("Generating election params...")
	// The election parameters are generated, including the start and end times, voting method, and choices.
	electionParams := generateElectionParams(elligibilityList)
	fmt.Println("Creating election...")
	// A mock secrets manager (mockSecretsManager) is created.
	secretsManager := new(mockSecretsManager)
	// A mock broadcast channel (MockBroadcastChannel)is created.
	broadcast := new(MockBroadcastChannel)
	// An Election instance is initialized with the previously generated components.
	var election Election
	broadcast.params = &electionParams
	election.credSys = credSys
	election.channel = broadcast
	election.secrets = secretsManager
	election.vdf = &vdf.PietrzakVdf{MaxDifficulty: 1000000, DifficultyConversion: 10000}
	election.method = &methods.PluralityVoting{}
	election.params = &electionParams
	secretCredentials, err := generateSecretCredentials(credSys, len(privateKeys))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	// Iterates over the private keys and sets the current private key and secret credential in the secrets manager.
	for i := range privateKeys {
		secretsManager.privateKey = privateKeys[i]
		secretsManager.secretCredential = secretCredentials[i]
		// Post the credential to the broadcast channel.
		// This step ensures that all participants have posted their credentials.
		err = election.PostCredential(ctx)
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
	}
	// The test waits until the current time reaches the cast phase start time specified in the election parameters.
	for time.Now().Before(electionParams.CastStart) {
		// It sleeps in 1-second intervals to simulate the passage of time.
		time.Sleep(time.Second)
	}
	// Once the cast phase starts, the test randomly selects a voter by generating a random index within the range of private keys.
	voterIdx := rand.Intn(len(privateKeys))
	// The secrets manager's secret credential is set to the selected voter's credential.
	secretsManager.secretCredential = secretCredentials[voterIdx]
	fmt.Println("Voter", voterIdx)
	fmt.Println("Voting...")
	// The Vote method of the Election instance is called with a randomly chosen choice index to cast a vote.
	err = election.Vote(ctx, rand.Intn(len(electionParams.Choices)))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	// The test waits until the current time reaches the tally phase start time specified in the election parameters.
	for time.Now().Before(electionParams.TallyStart) {
		time.Sleep(time.Second)
	}
	fmt.Println("Tallying...")
	// The RevealBallotDecryption method of the Election instance is called to post the ballot decryption message to the broadcast channel.
	// perform ballot decryption using the VDF solution.
	err = election.RevealBallotDecryption(ctx)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	// The test waits until the current time reaches the tally phase end time specified in the election parameters.
	for time.Now().Before(electionParams.TallyEnd) {
		time.Sleep(time.Second)
	}
	fmt.Println("Progressing...")
	// Once the tally phase ends, the test retrieves the election progress using the Progress method of the Election instance.
	_, err = election.Progress(ctx)
	// The election progress includes the current phase, the count of valid ballots, the total number of participants, and the tally results.
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	fmt.Println("Done!")
}
