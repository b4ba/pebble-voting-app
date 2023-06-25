package voting

import (
	"bytes"
	"io"
	"net/http"

	"github.com/giry-dev/pebble-voting-app/pebble-core/util"
	"github.com/giry-dev/pebble-voting-app/pebble-core/voting/structs"
)

// Represents a client for interacting with a broadcast server.
// Contains an HTTP client and the URIs for retrieving election parameters and messages from the server.
type BroadcastClient struct {
	client                 http.Client
	paramsURI, messagesURI string
}

/*
Sends an HTTP GET request to the server's params URI.
Retrieves the response body and reads it into a byte buffer.
Creates a new ElectionParams struct and populates it by calling the FromBytes() method, passing the byte buffer as the input.
Returns the populated ElectionParams struct or an error if there was a problem retrieving or parsing the response.
*/
func (bc *BroadcastClient) Params() (*ElectionParams, error) {
	resp, err := bc.client.Get(bc.paramsURI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	p := new(ElectionParams)
	err = p.FromBytes(buf)
	if err != nil {
		return nil, err
	}
	return p, nil
}

/*
Sends an HTTP GET request to the server's messages URI.
Retrieves the response body and reads it into a byte buffer.
Creates a new util.BufferReader and initializes an empty slice of Message structs.
Parses the byte buffer to extract individual messages by reading the message kind (represented by a byte) and the message bytes.
Based on the message kind, creates a new structs.CredentialMessage, structs.SignedBallot, or structs.DecryptionMessage and populates it by calling the respective FromBytes() method.
Appends the populated message to the slice of Message structs.
Returns the slice of Message structs or an error if there was a problem retrieving or parsing the response.
*/
func (bc *BroadcastClient) Get() ([]Message, error) {
	resp, err := bc.client.Get(bc.messagesURI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	r := util.NewBufferReader(buf)
	var msgs []Message
	for r.Len() != 0 {
		kind, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		m, err := r.ReadVector()
		if err != nil {
			return nil, err
		}
		switch ElectionPhase(kind) {
		case CredGen:
			msg := new(structs.CredentialMessage)
			err = msg.FromBytes(m)
			if err == nil {
				msgs = append(msgs, Message{Credential: msg})
			}
		case Cast:
			msg := new(structs.SignedBallot)
			err = msg.FromBytes(m)
			if err == nil {
				msgs = append(msgs, Message{SignedBallot: msg})
			}
		case Tally:
			msg := new(structs.DecryptionMessage)
			err = msg.FromBytes(m)
			if err == nil {
				msgs = append(msgs, Message{Decryption: msg})
			}
		}
	}
	return msgs, nil
}

/*
Sends an HTTP POST request to the server's messages URI.
Creates a byte buffer from the input Message by calling the Bytes() method.
Sends the byte buffer as the request body with the content type set to "application/octet-stream".
Returns an error if there was a problem sending the request or closing the response body.
*/
func (bc *BroadcastClient) Post(m Message) error {
	resp, err := bc.client.Post(bc.messagesURI, "application/octet-stream", bytes.NewReader(m.Bytes()))
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
