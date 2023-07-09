package voting

import (
	"context"
	"errors"
	"fmt"

	"github.com/giry-dev/pebble-voting-app/pebble-core/voting/secrets"
)

var (
	ErrUnknownNetwork = errors.New("pebble: unknown network")
	ErrNoServers      = errors.New("pebble: no servers in invitation")
	ErrInvalidAddress = errors.New("pebble: invalid address")
)

func NewElectionFromInvitation(ctx context.Context, inv Invitation, sec secrets.SecretsManager) (*Election, error) {
	fmt.Println("Creating election from invitation...")
	switch inv.Network {
	case "mock":
		if len(inv.Servers) == 0 {
			return nil, ErrNoServers
		}
		bc, err := NewBroadcastClient(string(inv.Address), inv.Servers[0])
		fmt.Println("Broadcast client created...")
		election_params, err := bc.Params(ctx)

		fmt.Println("Params retrieved...")
		fmt.Println(election_params.Version)
		fmt.Println(election_params.Title)
		fmt.Println(election_params.Description)
		fmt.Println(election_params.CastStart)
		fmt.Println(election_params.TallyStart)
		fmt.Println(election_params.TallyEnd)
		fmt.Println(election_params.MaxVdfDifficulty)
		fmt.Println(election_params.VotingMethod)
		fmt.Println(election_params.Choices)
		fmt.Println(election_params.EligibilityList)
		fmt.Println("Params printed...")
		if err != nil {
			return nil, err
		}
		return NewElection(ctx, bc, sec)
	default:
		fmt.Println("Unknown network...")
		return nil, ErrUnknownNetwork
	}
}
