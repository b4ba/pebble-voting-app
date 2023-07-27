package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/giry-dev/pebble-voting-app/pebble-core/base32c"
	"github.com/giry-dev/pebble-voting-app/pebble-core/util"
	"github.com/giry-dev/pebble-voting-app/pebble-core/voting"
	"github.com/giry-dev/pebble-voting-app/pebble-core/voting/secrets"
)

type handler struct {
	file      secrets.FileSecretsManager
	elections map[string]*voting.Election
}

func main() {
	// port := rand.Intn(40000) + 10000
	// endpoint := "127.0.0.1:" + strconv.Itoa(port)
	// endpoint := "localhost:" + strconv.Itoa(port)
	endpoint := "127.0.0.1:8080"
	handler := &handler{
		file:      "secrets.json",
		elections: make(map[string]*voting.Election),
	}
	fmt.Printf("http://%s\n", endpoint)
	err := http.ListenAndServe(endpoint, handler)
	if err != nil {
		fmt.Println(err)
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	projectRoot := "/Users/alexbabalitis/Documents/EDINBURGH_UNI/Dissertation/pebble-voting-app"

	path := req.URL.Path
	if path == "/" {
		indexPath := filepath.Join(projectRoot, "basic-web-ui", "index.html")
		indexFile, err := os.ReadFile(indexPath)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		w.Header().Add("Content-Type", "text/html")
		w.Header().Add("Content-Length", strconv.Itoa(len(indexFile)))
		w.WriteHeader(200)
		w.Write(indexFile)
		return
	} else if path == "/style.css" {
		stylePath := filepath.Join(projectRoot, "basic-web-ui", "style.css")
		styleFile, err := os.ReadFile(stylePath)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		w.Header().Add("Content-Type", "text/css")
		w.Header().Add("Content-Length", strconv.Itoa(len(styleFile)))
		w.WriteHeader(200)
		w.Write(styleFile)
		return
	} else if path == "/script.js" {
		scriptPath := filepath.Join(projectRoot, "basic-web-ui", "script.js")
		scriptFile, err := os.ReadFile(scriptPath)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		w.Header().Add("Content-Type", "application/javascript")
		w.Header().Add("Content-Length", strconv.Itoa(len(scriptFile)))
		w.WriteHeader(200)
		w.Write(scriptFile)
		return
	} else if strings.HasPrefix(path, "/api/") {
		if path == "/api/pubkey" {
			priv, err := h.file.GetPrivateKey(nil)
			if err != nil {
				http.Error(w, "Error reading private key", http.StatusInternalServerError)
				return
			}
			str, err := priv.Public().String()
			if err != nil {
				http.Error(w, "Error reading private key", http.StatusInternalServerError)
				return
			}
			respondText(w, http.StatusOK, str)
			return
		}
		ctx := context.Background()
		if invStr, ok := util.GetSuffix(path, "/api/election/join/"); ok {
			inv, err := voting.DecodeInvitation(invStr)
			if err != nil {
				http.Error(w, fmt.Sprint("Error decoding invitation:", err), http.StatusBadRequest)
				return
			}
			election, err := voting.NewElectionFromInvitation(ctx, inv, h.file)
			fmt.Println("Election:", election)
			if err != nil {
				http.Error(w, fmt.Sprint("Error joining election:", err), http.StatusInternalServerError)
				return
			}
			fmt.Println("Moving to post credential commitment")
			err = election.PostCredentialCommitment(ctx)
			if err != nil {
				http.Error(w, fmt.Sprint("Error posting credential commitment:", err), http.StatusInternalServerError)
				return
			}
			eid := election.Id()
			err = h.file.SetElection(secrets.BasicElectionInfo{
				Id:         base32c.Encode(eid[:]),
				Invitation: invStr,
				Title:      election.Params().Title,
			})
			if err != nil {
				http.Error(w, fmt.Sprint("Error writing election info:", err), http.StatusInternalServerError)
				return
			}
			h.elections[invStr] = election
			respondText(w, http.StatusOK, "Joined election")
			fmt.Println("Successfully joined election!!!")
			return
		} else if invStr, ok := util.GetSuffix(path, "/api/election/info/"); ok {
			election, err := h.election(ctx, invStr)
			if err != nil {
				http.Error(w, "Error getting election info:", http.StatusInternalServerError)
			}
			params := election.Params()
			var resp struct {
				Title       string   `json:"title"`
				Description string   `json:"description"`
				CastStart   string   `json:"castStart"`
				TallyStart  string   `json:"tallyStart"`
				Choices     []string `json:"choices"`
			}
			resp.Title = params.Title
			resp.Description = params.Description
			resp.CastStart = params.CastStart.Format(time.RFC3339)
			resp.TallyStart = params.TallyStart.Format(time.RFC3339)
			resp.Choices = params.Choices

			fmt.Println("Election info:", resp)
			respondJson(w, resp)
			return
		}
	}
	http.Error(w, "Not Found", http.StatusNotFound)
}

func respondText(w http.ResponseWriter, statusCode int, body string) {
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(statusCode)
	w.Write([]byte(body))
}

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

func (h *handler) election(ctx context.Context, invStr string) (*voting.Election, error) {
	if election, ok := h.elections[invStr]; ok {
		return election, nil
	}
	inv, err := voting.DecodeInvitation(invStr)
	if err != nil {
		return nil, err
	}
	election, err := voting.NewElectionFromInvitation(ctx, inv, h.file)
	if err != nil {
		return nil, err
	}
	eid := election.Id()
	err = h.file.SetElection(secrets.BasicElectionInfo{
		Id:         base32c.Encode(eid[:]),
		Invitation: invStr,
		Title:      election.Params().Title,
	})
	if err != nil {
		return nil, err
	}
	h.elections[invStr] = election
	return election, nil
}
