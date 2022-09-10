package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	main "github.com/diceride/go-workflow-example"
	"github.com/diceride/go-workflow-example/workflow"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

type TestifySuite struct {
	s main.Service
	suite.Suite
}

func (ts *TestifySuite) SetupTest() {
	cadenceConfig := main.CadenceConfig{
		Addr:         "127.0.0.1:7933",
		ClientName:   "cadence-client",
		ServiceName:  "cadence-frontend",
		Domain:       "test-domain",
		TaskListName: "test-tasks",
		WorkflowName: "test",
	}

	workflowClient, err := main.InitCadenceWorker(cadenceConfig)
	if err != nil {
		log.Fatal(err)
	}

	s := main.Service{
		WorkflowClient: workflowClient,
		CadenceConfig:  &cadenceConfig,
	}

	ts.s = s
}

func TestTestifySuite(t *testing.T) {
	suite.Run(t, new(TestifySuite))
}

func (ts *TestifySuite) Test_Workflow() {
	id := uuid.New()

	var jsonData = []byte(fmt.Sprintf(`{
		"name": "%s",
		"waitingTime": 30
	}`, id))

	r, _ := http.NewRequest("POST", "/workflow", bytes.NewBuffer(jsonData))
	w := httptest.NewRecorder()

	ts.s.Handler(w, r)

	resp := w.Result()

	ts.Equal(http.StatusAccepted, resp.StatusCode)
	ts.Equal("/workflow/status", resp.Header.Get("Location"))
	ts.Equal("1", resp.Header.Get("Retry-After"))

	data := make(map[string]interface{})

	err := json.NewDecoder(resp.Body).Decode(&data)
	ts.Nil(err)

	ts.Equal(data["id"], id)

	path := ts.waitForWorkflow(url.URL{
		Path:     resp.Header.Get("Location"),
		RawQuery: "id=" + id,
	})

	ts.Equal("/workflow/result", path)

	reqUrl := url.URL{
		Path:     path,
		RawQuery: "id=" + id,
	}

	r, _ = http.NewRequest("GET", reqUrl.String(), bytes.NewBuffer(jsonData))
	w = httptest.NewRecorder()

	ts.s.Handler(w, r)

	resp = w.Result()

	ts.Equal(http.StatusOK, resp.StatusCode)

	data = make(map[string]interface{})

	err = json.NewDecoder(resp.Body).Decode(&data)
	ts.Nil(err)

	ts.Equal(workflow.State_name[workflow.State_COMPLETED], data["status"])
}

func (ts *TestifySuite) waitForWorkflow(reqUrl url.URL) string {
	attempts := 0
	for {
		time.Sleep(time.Second)

		r, _ := http.NewRequest("GET", reqUrl.String(), &bytes.Buffer{})
		w := httptest.NewRecorder()

		ts.s.Handler(w, r)

		resp := w.Result()

		if resp.StatusCode == http.StatusFound {
			return resp.Header.Get("Location")
		}

		attempts++

		if attempts > 30 {
			log.Panic("timeout")
		}
	}
}
