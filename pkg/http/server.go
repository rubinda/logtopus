package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rubinda/logtopus/pkg/influxdb"
	"golang.org/x/sync/errgroup"
)

const (
	// apiBasePath contains the prefix for each API endpoint.
	apiBasePath string = "/api/v1"
)

// TODO:
//   - username and password for testing authentication, could be replaced with OAuth or similar.
const (
	authorizedUsername string = "johnnyHotbody"
	authorizedPassword string = "me-llamo-johnny"
)

// Configuration contains required parameters to start a HTTP(S) server.
type Configuration struct {
	DB      *influxdb.Client
	Address string

	JWTKeyPath    string
	JWTPubKeyPath string
}

// Server contains methods to handle HTTP requests.
type Server struct {
	instance *http.Server
	db       *influxdb.Client
	jwtAuth  *JWTAuthority
}

// NewServer creates a new HTTP(S) server with the given parameters.
func NewServer(c Configuration, jwtAuth *JWTAuthority) *Server {
	s := &Server{db: c.DB, jwtAuth: jwtAuth}
	mux := http.NewServeMux()
	mux.HandleFunc(apiBasePath+"/auth", s.authHandler)
	mux.HandleFunc(apiBasePath+"/events", authMiddleware(jwtAuth, s.eventsHandler))
	mux.HandleFunc(apiBasePath+"/query/events", authMiddleware(jwtAuth, s.eventsQueryHandler))
	s.instance = &http.Server{
		Addr:         c.Address,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s
}

// Start runs the server instance to listen for incoming connections. Waits for CTRL-C to initiate a graceful shutdown.
func (server *Server) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		osSignals := make(chan os.Signal, 1)
		signal.Notify(osSignals, syscall.SIGINT, syscall.SIGTERM)
		<-osSignals
		cancel()
	}()

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		log.Printf("Server listening on http://%s\n", server.instance.Addr)
		return server.instance.ListenAndServe()
	})
	g.Go(func() error {
		<-gCtx.Done()
		log.Printf("Server shutdown request. \n")
		return server.Shutdown()
	})

	if err := g.Wait(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Exited: %s\n", err)
	}
}

// Shutdown gracefully terminates the http server and open DB connections.
func (server *Server) Shutdown() error {
	server.db.Disconnect()
	return server.instance.Shutdown(context.Background())
}

// authHandler authenticates an entity and responds with a token.
func (server *Server) authHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] %s\n", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodPost:
		loginInfo := struct {
			User string `json:"user"`
			Pass string `json:"pass"`
		}{}
		err := json.NewDecoder(r.Body).Decode(&loginInfo)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, errResponse{err.Error(), nil})
			return
		}
		if loginInfo.User != authorizedUsername || loginInfo.Pass != authorizedPassword {
			jsonResponse(w, http.StatusUnauthorized, errResponse{"Invalid username / password combination", nil})
			return
		}
		token, err := server.jwtAuth.issueToken(loginInfo.User)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, errResponse{"An error occurred while issuing your token. Please contact an administrator.", nil})
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"token": token})
	default:
		server.methodNotAllowed(w)
	}
}

// eventsHandler handles the "/events" API endpoint requests.
func (server *Server) eventsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] %s\n", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodPost:
		server.handleEventsPost(w, r)
	default:
		server.methodNotAllowed(w)
	}
}

// handleEventsPost handles the POST request on the "/events" endpoint.
func (server *Server) handleEventsPost(w http.ResponseWriter, r *http.Request) {
	var eventData influxdb.BasicEvent
	// Ensure proper JSON structure
	err := json.NewDecoder(r.Body).Decode(&eventData)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, errResponse{err.Error(), nil})
		return
	}
	// Ensure required fields
	if problems := eventData.Validate(); len(problems) > 0 {
		jsonResponse(w, http.StatusBadRequest, errResponse{errBadRequestBody, problems})
		return
	}
	// Store into database
	err = server.db.StoreEvent(eventData)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, errResponse{err.Error(), nil})
		return
	}
	// If we use non-blocking writing to the database (InfluxDB recommends batching for better performance),
	// the write operation status can't be determined at the time of the request.
	w.WriteHeader(http.StatusOK)
}

// eventsQueryHandler handles the "/query/events" API endpoint requests.
func (server *Server) eventsQueryHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] %s\n", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodPost:
		server.handleEventsQueryPost(w, r)
	default:
		server.methodNotAllowed(w)
	}
}

// handleEventsQuery handles POST requests on the "/query/events" endpoint.
func (server *Server) handleEventsQueryPost(w http.ResponseWriter, r *http.Request) {
	var queryFields map[string]any
	if err := json.NewDecoder(r.Body).Decode(&queryFields); err != nil && err != io.EOF {
		jsonResponse(w, http.StatusBadRequest, errResponse{err.Error(), nil})
		return
	}
	res, err := server.db.QueryEvents(queryFields)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, errResponse{err.Error(), nil})
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// methodNotAllowed writes the equally named HTTP status to given ResponseWriter.
func (server *Server) methodNotAllowed(w http.ResponseWriter) {
	jsonResponse(w, http.StatusMethodNotAllowed, errResponse{http.StatusText(http.StatusMethodNotAllowed), nil})
}

// jsonResponse returns the given body and status code to the client as a JSON document.
func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	b, err := json.Marshal(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Write(b)
}
