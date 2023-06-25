package voting

import (
	"context"
	"errors"

	"github.com/giry-dev/pebble-voting-app/pebble-core/anoncred"
	"github.com/giry-dev/pebble-voting-app/pebble-core/util"
	"github.com/giry-dev/pebble-voting-app/pebble-core/vdf"
	"github.com/giry-dev/pebble-voting-app/pebble-core/voting/methods"
	"github.com/giry-dev/pebble-voting-app/pebble-core/voting/secrets"
	"github.com/giry-dev/pebble-voting-app/pebble-core/voting/structs"
)

var (
	ErrWrongPhase = errors.New("pebble: wrong election phase")

	ErrDecryptionNotFound = errors.New("pebble: ballot decryption not found")
)

type ElectionID = [32]byte

// Represents an election and contains various components and parameters related to the election
type Election struct {
	credSys anoncred.CredentialSystem
	channel BroadcastChannel
	secrets secrets.SecretsManager
	vdf     vdf.VDF
	method  methods.VotingMethod
	params  *ElectionParams
}

// Represents the progress of an election, including the current phase,
// the count and total number of processed items, and the tally (if applicable).
type ElectionProgress struct {
	Phase        ElectionPhase
	Count, Total int
	Tally        methods.Tally
}

/*
Creates a new Election instance.
Initializes the credential system, voting method, VDF, and other components based on the provided broadcast channel and secrets manager.
Retrieves the election parameters from the broadcast channel.
Returns the created Election instance or an error.
*/
func NewElection(ctx context.Context, bc BroadcastChannel, sec secrets.SecretsManager) (*Election, error) {
	if anoncred.AnonCred1Instance == nil {
		return nil, errors.New("pebble: anoncred.AnonCred1Instance is nil")
	}
	params, err := bc.Params(ctx)
	if err != nil {
		return nil, err
	}
	method, err := methods.Get(params.VotingMethod, len(params.Choices))
	if err != nil {
		return nil, err
	}
	vdf := &vdf.PietrzakVdf{
		MaxDifficulty:        params.MaxVdfDifficulty,
		DifficultyConversion: uint64(float64(params.MaxVdfDifficulty) / params.TallyStart.Sub(params.CastStart).Seconds()),
	}
	return &Election{
		credSys: anoncred.AnonCred1Instance,
		channel: bc,
		secrets: sec,
		vdf:     vdf,
		method:  method,
		params:  params,
	}, nil
}

// Returns the election parameters of the Election instance.
func (e *Election) Params() *ElectionParams {
	return e.params
}

//  Returns the current phase of the election.
func (e *Election) Phase() ElectionPhase {
	return e.params.Phase()
}

// Returns the ID of the election.
func (e *Election) Id() ElectionID {
	return e.channel.Id()
}

// Returns the broadcast channel associated with the election.
func (e *Election) Channel() BroadcastChannel {
	return e.channel
}

/*
Posts the credential message to the broadcast channel.
Checks if the current phase of the election allows posting credentials.
Retrieves the private key and secret credential from the secrets manager.
Creates and signs the credential message.
Posts the message to the broadcast channel.
Returns an error if the phase is incorrect or any step fails.
*/
func (e *Election) PostCredential(ctx context.Context) error {
	if e.params.Phase() != CredGen {
		return ErrWrongPhase
	}
	priv, err := e.secrets.GetPrivateKey()
	if err != nil {
		return err
	}
	sec, err := e.secrets.GetSecretCredential(e.credSys)
	if err != nil {
		return err
	}
	pub, err := sec.Public()
	if err != nil {
		return err
	}
	msg := new(structs.CredentialMessage)
	msg.Credential = pub.Bytes()
	err = msg.Sign(priv, e.Id())
	if err != nil {
		return err
	}
	return e.channel.Post(ctx, Message{Credential: msg})
}

/*
Retrieves the credential set from the broadcast channel.
Checks if the current phase of the election allows retrieving credentials.
Fetches the messages from the broadcast channel.
Verifies and reads the public credentials from the received messages.
Constructs the credential set using the credential system.
Returns the credential set or an error if the phase is incorrect or any step fails.
*/
func (e *Election) GetCredentialSet(ctx context.Context) (anoncred.CredentialSet, error) {
	if e.params.Phase() <= CredGen {
		return nil, ErrWrongPhase
	}
	msgs, err := e.channel.Get(ctx)
	if err != nil {
		return nil, err
	}
	creds := make(map[util.HashValue]anoncred.PublicCredential)
	for _, msg := range msgs {
		if msg.Credential == nil {
			continue
		}
		if msg.Credential.Verify(e.Id()) != nil {
			continue
		}
		cred, err := e.credSys.ReadPublicCredential(msg.Credential.Credential)
		if err != nil {
			continue
		}
		creds[util.Hash(msg.Credential.PublicKey)] = cred
	}
	var list []anoncred.PublicCredential
	for _, c := range creds {
		list = append(list, c)
	}
	return e.credSys.MakeCredentialSet(list)
}

/*
Casts a vote in the election.
Checks if the current phase of the election allows voting.
Retrieves the credential set.
Generates a VDF solution.
Encrypts the ballot using the VDF solution.
Signs the encrypted ballot using the credential set and secret credential.
Stores the signed ballot in the secrets manager.
Posts the signed ballot to the broadcast channel.
Returns an error if the phase is incorrect or any step fails.
*/
func (e *Election) Vote(ctx context.Context, choices ...int) error {
	if e.params.Phase() != Cast {
		return ErrWrongPhase
	}
	set, err := e.GetCredentialSet(ctx)
	if err != nil {
		return err
	}
	sol, err := e.vdf.Create(e.puzzleDuration())
	if err != nil {
		return err
	}
	err = e.secrets.SetVdfSolution(sol)
	if err != nil {
		return err
	}
	sec, err := e.secrets.GetSecretCredential(e.credSys)
	if err != nil {
		return err
	}
	ballot := e.method.Vote(choices...)
	encBallot, err := ballot.Encrypt(sol)
	if err != nil {
		return err
	}
	signBallot, err := encBallot.Sign(set, sec)
	if err != nil {
		return err
	}
	err = e.secrets.SetBallot(signBallot)
	if err != nil {
		return err
	}
	return e.channel.Post(ctx, Message{SignedBallot: &signBallot})
}

func (e *Election) puzzleDuration() uint64 {
	// Calculates the duration of the puzzle (VDF) based on the election parameters.
	// Returns the puzzle duration as a uint64 value.
	return uint64(e.params.TallyStart.Sub(e.params.CastStart).Seconds())
}

/*
Retrieves the VDF solution from the secrets manager.
Calls PostBallotDecryption with the VDF solution as the parameter.
Returns an error if the VDF solution retrieval or posting fails.
*/
func (e *Election) RevealBallotDecryption(ctx context.Context) error {
	sol, err := e.secrets.GetVdfSolution()
	if err != nil {
		return err
	}
	return e.PostBallotDecryption(ctx, sol)
}

/*
Posts the ballot decryption message to the broadcast channel.
Checks if the current phase of the election allows posting ballot decryption.
Creates the decryption message using the provided VDF solution.
Posts the message to the broadcast channel.
Returns an error if the phase is incorrect or any step fails.
*/
func (e *Election) PostBallotDecryption(ctx context.Context, sol vdf.VdfSolution) error {
	if e.params.Phase() != Tally {
		return ErrWrongPhase
	}
	msg := structs.CreateDecryptionMessage(sol)
	return e.channel.Post(ctx, Message{Decryption: &msg})
}

/*
Retrieves the progress of the election.
Determines the current phase of the election.
Retrieves the credential set and messages from the broadcast channel.
Processes the signed ballots and decryption messages to calculate the progress.
Returns an ElectionProgress struct with the phase, count, total, and tally (if applicable), or an error.
*/
func (e *Election) Progress(ctx context.Context) (p ElectionProgress, err error) {
	p.Phase = e.params.Phase()
	if p.Phase <= CredGen {
		return
	}
	set, err := e.GetCredentialSet(ctx)
	if err != nil {
		return
	}
	msgs, err := e.channel.Get(ctx)
	if err != nil {
		return
	}
	var signBallots []structs.SignedBallot
	var decMsgs []structs.DecryptionMessage
	for _, msg := range msgs {
		if msg.SignedBallot != nil {
			signBallots = append(signBallots, *msg.SignedBallot)
		} else if msg.Decryption != nil {
			decMsgs = append(decMsgs, *msg.Decryption)
		}
	}
	var serialNos util.BytesSet
	var decBallots []structs.Ballot
	validSignBallots := 0
	validDecBallots := 0
	invalidDecBallots := 0
	for _, signBallot := range signBallots {
		if serialNos.Contains(signBallot.SerialNo) {
			continue
		}
		err = signBallot.Verify(set)
		if err != nil {
			continue
		}
		validSignBallots++
		if p.Phase >= Tally {
			ballot, err := decryptBallot(signBallot.EncryptedBallot, decMsgs, e.vdf)
			if err != nil {
				if err != ErrDecryptionNotFound {
					invalidDecBallots++
				}
				continue
			}
			decBallots = append(decBallots, ballot)
			validDecBallots++
		}
	}
	if p.Phase == Cast {
		p.Total = set.Len()
		p.Count = validSignBallots
	} else if p.Phase == Tally {
		p.Total = validSignBallots - invalidDecBallots
		p.Count = validDecBallots
		p.Tally = e.method.Tally(decBallots)
	} else {
		p.Total = validSignBallots
		p.Count = validDecBallots
		p.Tally = e.method.Tally(decBallots)
	}
	return p, nil
}

/*
Decrypts an encrypted ballot using the provided decryption messages and VDF.
Takes the encrypted ballot, decryption messages, and VDF as input.
Checks if the VDF solution matches the input hash of the encrypted ballot.
Verifies the VDF solution.
Decrypts the ballot using the VDF solution.
Returns the decrypted ballot or an error if the decryption is not found or fails.
*/
func decryptBallot(encBallot structs.EncryptedBallot, msgs []structs.DecryptionMessage, ivdf vdf.VDF) (structs.Ballot, error) {
	vdfInputHash := util.Hash(encBallot.VdfInput)
	for _, msg := range msgs {
		if msg.InputHash == vdfInputHash {
			sol := vdf.VdfSolution{Input: encBallot.VdfInput, Output: msg.Output, Proof: msg.Proof}
			err := ivdf.Verify(sol)
			if err != nil {
				continue
			}
			ballot, err := encBallot.Decrypt(sol)
			if err != nil {
				return nil, err
			}
			return ballot, nil
		}
	}
	return nil, ErrDecryptionNotFound
}
