package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/diceride/go-workflow-example/workflow"
	cadenceclient "go.uber.org/cadence/client"
)

type WorkflowExecutionReference struct {
	id    string
	state workflow.State
}

type Service struct {
	WorkflowClient *cadenceclient.Client
	CadenceConfig  *CadenceConfig
}

// HTTP handler.
func (s *Service) Handler(w http.ResponseWriter, r *http.Request) {
	// Disable client caching
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))

	// CORS preflight handler
	if r.Method == "OPTIONS" {
		w.Header().Add("Connection", "keep-alive")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "OPTIONS, GET, POST")
		w.Header().Add("Access-Control-Allow-Headers", "content-type")
		return
	}

	// Match the URL path
	switch r.URL.Path {
	case "/workflow":
		switch r.Method {
		case "POST":
			s.createWorkflowHandler(w, r)
		default:
			w.Header().Set("Allow", "POST")
			http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		}
	case "/workflow/status":
		switch r.Method {
		case "GET":
			s.workflowStatusHandler(w, r)
		default:
			w.Header().Set("Allow", "GET")
			http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		}
	case "/workflow/result":
		switch r.Method {
		case "GET":
			s.workflowResultHandler(w, r)
		default:
			w.Header().Set("Allow", "GET")
			http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.NotFound(w, r)
	}
}

// HTTP POST /workflow handler.
func (s *Service) createWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		// The unique name of the workflow
		Name string `json:"name"`
		// The waiting time in seconds
		WaitingTime time.Duration `json:"waitingTime"`
	}

	// Declare a new Request struct
	var data Request

	// Decode the request body into the Request struct
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	waitingTime := data.WaitingTime * time.Second

	if waitingTime < time.Duration(30*time.Second) {
		http.Error(w, "400 bad request", http.StatusBadRequest)
		return
	}

	// Create a new workflow
	exec, err := s.createWorkflow(data.Name, waitingTime)
	if err != nil {
		log.Printf("failed to create workflow: %v\n", err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/workflow/status")
	// Delay in seconds
	w.Header().Set("Retry-After", "1")
	w.WriteHeader(http.StatusAccepted)

	type Response struct {
		// The workflow execution ID
		ID string `json:"id"`
	}

	res := Response{
		ID: exec.id,
	}

	// Encode the Response struct into the body
	_ = json.NewEncoder(w).Encode(res)
}

// HTTP GET /workflow/status handler.
func (s *Service) workflowStatusHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "400 bad request", http.StatusBadRequest)
		return
	}

	// Find a workflow by its id
	exec, err := s.findWorkflowById(id)
	if err != nil {
		log.Printf("failed to find workflow: id \"%s\": %v\n", id, err)

		http.NotFound(w, r)
		return
	}

	if exec.state == workflow.State_STARTED {
		w.WriteHeader(http.StatusOK)
	} else {
		w.Header().Add("Location", "/workflow/result")
		w.WriteHeader(http.StatusFound)
	}
}

// HTTP GET /workflow/result handler.
func (s *Service) workflowResultHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "400 bad request", http.StatusBadRequest)
		return
	}

	// Find a workflow by its id
	exec, err := s.findWorkflowById(id)
	if err != nil {
		log.Printf("failed to find workflow: id \"%s\": %v\n", id, err)

		http.NotFound(w, r)
		return
	}

	type Response struct {
		// The workflow execution status
		Status string `json:"status"`
	}

	res := Response{
		Status: workflow.State_name[exec.state],
	}

	w.WriteHeader(http.StatusOK)

	// Encode the Response struct into the body
	_ = json.NewEncoder(w).Encode(res)
}

// Create a new workflow.
func (s *Service) createWorkflow(name string, waitingTime time.Duration) (*WorkflowExecutionReference, error) {
	workflowOptions := cadenceclient.StartWorkflowOptions{
		ID:                           name,
		TaskList:                     s.CadenceConfig.TaskListName,
		ExecutionStartToCloseTimeout: waitingTime + time.Duration(time.Second),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	exec, err := (*s.WorkflowClient).StartWorkflow(ctx, workflowOptions, workflow.StartWorkflow, waitingTime)
	if err != nil {
		return nil, err
	}

	return &WorkflowExecutionReference{
		id:    exec.ID,
		state: workflow.State_STARTED,
	}, nil
}

// Find a workflow by its id.
func (s *Service) findWorkflowById(id string) (*WorkflowExecutionReference, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	value, err := (*s.WorkflowClient).QueryWorkflow(ctx, id, "", "state")
	if err != nil {
		return nil, err
	}

	var state workflow.State
	err = value.Get(&state)
	if err != nil {
		return nil, err
	}

	return &WorkflowExecutionReference{
		id,
		state,
	}, nil
}
