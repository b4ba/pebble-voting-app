package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/giry-dev/pebble-voting-app/pebble-core/util"
	"github.com/giry-dev/pebble-voting-app/pebble-core/voting"
)

type SetupStatus uint8

// Defines three possible setup statuses for an election.
const (
	SetupError SetupStatus = iota
	SetupInProgress
	SetupDone
)

// Represents a server error and includes the status code and message.
type ServerError struct {
	StatusCode int
	Body       string
}

// It implements the Error() method to return a formatted error message.
func (e *ServerError) Error() string {
	return fmt.Sprintf("pebble: server error %d: %s", e.StatusCode, e.Body)
}

// Holds information about the setup status of an election.
type SetupInfo struct {
	Status     SetupStatus
	Error      string
	BackendId  string
	Invitation string
}

/*
Interface defines the methods for managing elections.
It includes methods like Create for creating an election,
Setup for retrieving setup information, and Election for getting an election instance.
*/
type ElectionService interface {
	Create(params ElectionSetupParams) error
	Setup(adminId string) SetupInfo
	Election(backendId string) (*voting.Election, error)
}

/*
Struct represents the server and holds an instance of ElectionService,
a password hash, and boolean flags create and post indicating whether
the server is allowed to create elections and post messages, respectively.
*/
type Server struct {
	srv          ElectionService
	passHash     []byte
	create, post bool
}

// Utility function that sends a plain text response with the given status code and body.
func respondText(w http.ResponseWriter, statusCode int, body string) {
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(statusCode)
	w.Write([]byte(body))
}

// Utility function that marshals an object to JSON and sends it as the response with the appropriate content type.
func respondJson(w http.ResponseWriter, o interface{}) {
	content, err := json.Marshal(o)
	if err != nil {
		respondText(w, 500, err.Error())
	} else {
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Content-Length", strconv.Itoa(len(content)))
		w.WriteHeader(200)
		w.Write(content)
	}
}

// Decodes JSON data from the given reader into the provided interface.
func decodeJson(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

// Main handler for incoming HTTP requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	path := req.URL.Path
	/*
		/create (HTTP POST):

		Description: Create an election.
		Payload: JSON payload containing election setup parameters (ElectionSetupParams).
		Response: Plain text response indicating the status of the request.
	*/
	if path == "/create" {
		if req.Method != http.MethodPost {
			respondText(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		if !s.create {
			respondText(w, http.StatusForbidden, "Server does not create elections")
			return
		}
		if !s.authorized(w, req) {
			return
		}
		var params ElectionSetupParams
		err := decodeJson(req.Body, &params)
		if err != nil {
			respondText(w, 400, err.Error())
			return
		}
		err = s.srv.Create(params)
		if err != nil {
			respondText(w, 500, err.Error())
			return
		}
		respondText(w, 200, "Election creation enqueued")
		return

		/*
			/setup/{adminId} (HTTP GET):

			Description: Get setup information for an election.
			Parameters: adminId - The admin ID associated with the election setup.
			Response: JSON object containing the setup status (SetupInfo) with fields such as status, message, backend ID, and invitation.
		*/
	} else if adminId, ok := util.GetSuffix(path, "/setup/"); ok {
		if req.Method != http.MethodGet {
			respondText(w, 405, "Method not allowed")
			return
		}
		if !s.create {
			respondText(w, http.StatusForbidden, "Server does not create elections")
			return
		}
		if !s.authorized(w, req) {
			return
		}
		info := s.srv.Setup(adminId)
		var resp struct {
			Status     string `json:"status"`
			Message    string `json:"message,omitempty"`
			BackendId  string `json:"backendId,omitempty"`
			Invitation string `json:"invitation,omitempty"`
		}
		switch info.Status {
		case SetupError:
			resp.Status = "SetupError"
			resp.Message = info.Error
		case SetupInProgress:
			resp.Status = "InProgress"
		case SetupDone:
			resp.Status = "Done"
			resp.BackendId = info.BackendId
			resp.Invitation = info.Invitation
		default:
			respondText(w, 500, "Unknown status")
			return
		}
		respondJson(w, resp)

		/*
			/election/{backendId} (HTTP GET):

			Description: Get the status of an election.
			Parameters: backendId - The backend ID associated with the election.
			Response: JSON object representing the status of the election with fields specific to the progress phase of the election (voting.Election).
		*/
	} else if backendId, ok := util.GetSuffix(path, "/election/"); ok {
		if req.Method != http.MethodGet {
			respondText(w, 405, "Method not allowed")
			return
		}
		election, err := s.srv.Election(backendId)
		if err != nil {
			respondText(w, 500, err.Error())
			return
		}
		prog, err := election.Progress(ctx)
		if err != nil {
			respondText(w, 500, err.Error())
			return
		}
		switch prog.Phase {
		case voting.Setup:
			var resp struct {
				Status string `json:"status"`
			}
			resp.Status = "Setup"
			respondJson(w, resp)
		case voting.CredGen:
			var resp struct {
				Status string `json:"status"`
			}
			resp.Status = "CredGen"
			respondJson(w, resp)
		case voting.Cast:
			var resp struct {
				Status   string `json:"status"`
				Progress int    `json:"progress"`
				Total    int    `json:"total"`
			}
			resp.Status = "Cast"
			resp.Progress = prog.Count
			resp.Total = prog.Total
			respondJson(w, resp)
		case voting.Tally:
			var resp struct {
				Status   string         `json:"status"`
				Progress int            `json:"progress"`
				Total    int            `json:"total"`
				Counts   map[string]int `json:"counts"`
			}
			resp.Status = "Tally"
			respondJson(w, resp)
		case voting.End:
			var resp struct {
				Status string         `json:"status"`
				Valid  int            `json:"valid"`
				Total  int            `json:"total"`
				Counts map[string]int `json:"counts"`
			}
			resp.Status = "End"
			respondJson(w, resp)
		}

		/*
			/params/{backendId} (HTTP GET):

			Description: Get the parameters of an election.
			Parameters: backendId - The backend ID associated with the election.
			Response: Byte slice representing the serialized parameters of the election.
		*/
	} else if backendId, ok := util.GetSuffix(path, "/params/"); ok {
		if req.Method != http.MethodGet {
			respondText(w, 405, "Method not allowed")
			return
		}
		election, err := s.srv.Election(backendId)
		if err != nil {
			respondText(w, 500, err.Error())
			return
		}
		body := election.Params().Bytes()
		w.Header().Add("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(200)
		w.Write(body)

		/*
			/messages/{backendId} (HTTP GET and POST):

			Description: Get or post messages related to an election.
			Parameters: backendId - The backend ID associated with the election.
			GET Response: Byte slice representing the serialized messages retrieved from the election channel.
			POST Payload: Raw message bytes to be posted to the election channel.
			POST Response: Plain text response indicating the status of the message posting.
		*/
	} else if backendId, ok := util.GetSuffix(path, "/messages/"); ok {
		election, err := s.srv.Election(backendId)
		if err != nil {
			respondText(w, 500, err.Error())
			return
		}
		if req.Method == http.MethodGet {
			msgs, err := election.Channel().Get(ctx)
			if err != nil {
				respondText(w, 500, err.Error())
				return
			}
			w.WriteHeader(200)
			l := []byte{0, 0}
			for _, msg := range msgs {
				p := msg.Bytes()
				if len(p) < 128 {
					l[0] = byte(len(p))
					w.Write(l[:1])
				} else {
					l[0] = byte(len(p)>>8) | 128
					l[1] = byte(len(p))
					w.Write(l)
				}
				w.Write(p)
			}
		} else if req.Method == http.MethodPost {
			if !s.post {
				respondText(w, 403, "Server does not post messages")
				return
			}
			p, err := io.ReadAll(req.Body)
			if err != nil {
				respondText(w, 400, err.Error())
				return
			}
			msg, err := voting.MessageFromBytes(p)
			if err != nil {
				respondText(w, 400, err.Error())
				return
			}
			err = election.Channel().Post(ctx, msg)
			if err != nil {
				respondText(w, 500, err.Error())
			} else {
				respondText(w, 200, "Message posted")
			}
		} else {
			respondText(w, 405, "Method not allowed")
		}
	} else {
		respondText(w, 404, "Endpoint not found --> Default handler")
	}
}

// Checks if the request is authorized by comparing the provided password hash with the server's password hash.
func (s *Server) authorized(w http.ResponseWriter, req *http.Request) bool {
	if len(s.passHash) == 0 {
		return true
	}
	if _, pass, ok := req.BasicAuth(); ok {
		passHash := sha256.Sum256([]byte(pass))
		if subtle.ConstantTimeCompare(passHash[:], s.passHash) == 1 {
			return true
		}
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
	respondText(w, http.StatusUnauthorized, "Unauthorized")
	return false
}
