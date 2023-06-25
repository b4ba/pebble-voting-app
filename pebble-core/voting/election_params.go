/*
The election_params.go file provides the necessary functionality to serialize and deserialize the election parameters.
It allows the election parameters to be converted to a byte slice representation that can be stored or transmitted, and vice versa.
*/

package voting

import (
	"errors"
	"time"

	"github.com/giry-dev/pebble-voting-app/pebble-core/util"
	"github.com/giry-dev/pebble-voting-app/pebble-core/voting/structs"
)

var errUnknownVersion = errors.New("pebble: unknown ElectionParams version")

type ElectionPhase uint8

// Represents different phases of an election.
const (
	Setup ElectionPhase = iota
	CredGen
	Cast
	Tally
	End
)

type ElectionParams struct {
	Version                         uint32
	CastStart, TallyStart, TallyEnd time.Time
	MaxVdfDifficulty                uint64
	VotingMethod                    string
	Title, Description              string
	Choices                         []string
	EligibilityList                 *structs.EligibilityList
}

// Returns the current phase of the election based on the current time.
func (p *ElectionParams) Phase() ElectionPhase {
	now := time.Now()
	if now.Before(p.CastStart) {
		return CredGen
	} else if now.Before(p.TallyStart) {
		return Cast
	} else if now.Before(p.TallyEnd) {
		return Tally
	} else {
		return End
	}
}

/*
Serializes the ElectionParams struct into a byte slice.
Uses a BufferWriter from the util package to write each field in a specific order.
Converts time values to Unix timestamps and writes them as uint64.
Writes other fields as vectors of bytes.
Returns the serialized byte slice.
*/
func (p *ElectionParams) Bytes() []byte {
	var w util.BufferWriter
	w.WriteUint32(p.Version)
	w.WriteUint64(uint64(p.CastStart.Unix()))
	w.WriteUint64(uint64(p.TallyStart.Unix()))
	w.WriteUint64(uint64(p.TallyEnd.Unix()))
	w.WriteUint64(p.MaxVdfDifficulty)
	w.WriteVector([]byte(p.VotingMethod))
	w.WriteVector([]byte(p.Title))
	w.WriteVector([]byte(p.Description))
	w.WriteByte(byte(len(p.Choices)))
	for _, c := range p.Choices {
		w.WriteVector([]byte(c))
	}
	w.Write(p.EligibilityList.Bytes())
	return w.Buffer
}

/*
Deserializes a byte slice into an ElectionParams struct.
Uses a BufferReader from the util package to read the serialized byte slice.
Reads each field in the reverse order of serialization, extracting the values from the byte slice and assigning them to the corresponding struct fields.
Converts Unix timestamps back to time values.
Returns an error if any reading or conversion fails.
*/
func (p *ElectionParams) FromBytes(b []byte) (err error) {
	r := util.NewBufferReader(b)
	p.Version, err = r.ReadUint32()
	if err != nil {
		return err
	}
	if p.Version != 0 {
		return errUnknownVersion
	}
	if err != nil {
		return err
	}
	t, err := r.ReadUint64()
	if err != nil {
		return err
	}
	p.CastStart = time.Unix(int64(t), 0)
	t, err = r.ReadUint64()
	if err != nil {
		return err
	}
	p.TallyStart = time.Unix(int64(t), 0)
	t, err = r.ReadUint64()
	if err != nil {
		return err
	}
	p.TallyEnd = time.Unix(int64(t), 0)
	p.MaxVdfDifficulty, err = r.ReadUint64()
	if err != nil {
		return err
	}
	b, err = r.ReadVector()
	if err != nil {
		return err
	}
	p.VotingMethod = string(b)
	b, err = r.ReadVector()
	if err != nil {
		return err
	}
	p.Title = string(b)
	b, err = r.ReadVector()
	if err != nil {
		return err
	}
	p.Description = string(b)
	numChoices, err := r.ReadByte()
	if err != nil {
		return err
	}
	p.Choices = make([]string, numChoices)
	for i := range p.Choices {
		b, err = r.ReadVector()
		if err != nil {
			return err
		}
		p.Choices[i] = string(b)
	}
	p.EligibilityList = structs.NewEligibilityList()
	err = p.EligibilityList.FromBytes(r.ReadRemaining())
	return err
}
