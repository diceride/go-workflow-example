package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	cadenceclient "go.uber.org/cadence/client"
	"go.uber.org/cadence/worker"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/tchannel"
)

const (
	defaultPort = 8080

	// Environment variables
	portEnv                = "PORT"
	cadenceHostEnv         = "CADENCE_HOST"
	cadencePortEnv         = "CADENCE_PORT"
	cadenceDomainEnv       = "CADENCE_DOMAIN"
	cadenceTaskListNameEnv = "CADENCE_TASK_LIST_NAME"
	cadenceWorkflowNameEnv = "CADENCE_WORKFLOW_NAME"
)

type CadenceConfig struct {
	Addr         string
	ClientName   string
	Domain       string
	TaskListName string
	ServiceName  string
	WorkflowName string
}

// Initializes a new cadence worker
func InitCadenceWorker(c CadenceConfig) (*cadenceclient.Client, error) {
	fmt.Println("connecting to cadence", c.Addr)

	// Create a new transport channel
	ch, err := tchannel.NewChannelTransport(tchannel.ServiceName(c.ClientName))
	if err != nil {
		return nil, fmt.Errorf("tchannel.NewChannelTransport: %v", err)
	}

	// Create a new yarpc dispatcher
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: c.ClientName,
		Outbounds: yarpc.Outbounds{
			c.ServiceName: {Unary: ch.NewSingleOutbound(c.Addr)},
		},
	})
	if err := dispatcher.Start(); err != nil {
		return nil, fmt.Errorf("dispatcher.Start: %v", err)
	}

	// Create a new cadence domain client
	domainClient := cadenceclient.NewDomainClient(
		workflowserviceclient.New(dispatcher.ClientConfig(c.ServiceName)),
		&cadenceclient.Options{})

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	_, err = domainClient.Describe(ctx, c.Domain)
	if err != nil {
		return nil, fmt.Errorf("domainClient.Describe: %v", err)
	}

	// Create a new cadence worker
	worker := worker.New(
		workflowserviceclient.New(dispatcher.ClientConfig(c.ServiceName)),
		c.Domain,
		c.TaskListName,
		worker.Options{})

	err = worker.Start()
	if err != nil {
		return nil, fmt.Errorf("worker.Start: %v", err)
	}

	// Create a new cadence client
	client := cadenceclient.NewClient(
		workflowserviceclient.New(dispatcher.ClientConfig(c.ServiceName)),
		c.Domain,
		&cadenceclient.Options{
			Identity: "test",
		})

	return &client, nil
}

func main() {
	// Get the `PORT` environment variable
	port, ok := os.LookupEnv(portEnv)
	if !ok {
		port = strconv.Itoa(defaultPort)
	}

	// Get the `CADENCE_HOST` environment variable
	cadenceHost, ok := os.LookupEnv(cadenceHostEnv)
	if !ok {
		log.Panicln("os.LookupEnv:", cadenceHostEnv)
	}

	// Get the `CADENCE_PORT` environment variable
	cadencePort, ok := os.LookupEnv(cadencePortEnv)
	if !ok {
		log.Panicln("os.LookupEnv:", cadencePortEnv)
	}

	// Get the `CADENCE_DOMAIN` environment variable
	cadenceDomain, ok := os.LookupEnv("CADENCE_DOMAIN")
	if !ok {
		log.Panicln("os.LookupEnv:", cadenceDomainEnv)
	}

	// Get the `CADENCE_TASK_LIST_NAME` environment variable
	cadenceTaskListName, ok := os.LookupEnv("CADENCE_TASK_LIST_NAME")
	if !ok {
		log.Panicln("os.LookupEnv:", cadenceTaskListNameEnv)
	}

	// Get the `CADENCE_WORKFLOW_NAME` environment variable
	cadenceWorkflowName, ok := os.LookupEnv("CADENCE_WORKFLOW_NAME")
	if !ok {
		log.Panicln("os.LookupEnv", cadenceWorkflowNameEnv)
	}

	cadenceConfig := CadenceConfig{
		Addr:         cadenceHost + ":" + cadencePort,
		ClientName:   "cadence-client",
		ServiceName:  "cadence-frontend",
		Domain:       cadenceDomain,
		TaskListName: cadenceTaskListName,
		WorkflowName: cadenceWorkflowName,
	}

	// Create a new cadence workflow client
	workflowClient, err := InitCadenceWorker(cadenceConfig)
	if err != nil {
		log.Fatalln(err)
	}

	// Create a new service instance
	svc := Service{
		WorkflowClient: workflowClient,
		CadenceConfig:  &cadenceConfig,
	}

	// Default handler
	http.HandleFunc("/", svc.Handler)

	// Create a new http server
	s := &http.Server{
		Addr: ":" + port,
	}

	// Create a new error channel
	errChannel := make(chan error, 1)

	// Create a new system call signal channel
	signalChannel := make(chan os.Signal, 1)

	// Bind system call events to the signal channel
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	// Run a blocking call in a separate goroutine and report errors via the error channel
	go func() {
		fmt.Println("listening on port", defaultPort)

		if err := s.ListenAndServe(); err != nil {
			errChannel <- err
		}
	}()

	// Block until either a system call signal, or server fatal error is received
	select {
	case err := <-errChannel:
		log.Fatalf("http.ListenAndServe: %v\n", err)
	case <-signalChannel:
		// Kubernetes default pod shutdown timeout is 30 seconds
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Attempt to gracefully shutdown the http server
		if err := s.Shutdown(ctx); err != nil {
			log.Fatalf("s.Shutdown: %v\n", err)
		} else {
			os.Exit(0)
		}
	}
}
