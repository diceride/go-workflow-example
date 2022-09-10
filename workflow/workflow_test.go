package workflow_test

import (
	"testing"
	"time"

	"github.com/diceride/go-workflow-example/workflow"
	"github.com/stretchr/testify/suite"
	cadencetestsuite "go.uber.org/cadence/testsuite"
)

func TestTestifySuite(t *testing.T) {
	suite.Run(t, new(TestifySuite))
}

type TestifySuite struct {
	env *cadencetestsuite.TestWorkflowEnvironment
	cadencetestsuite.WorkflowTestSuite
	suite.Suite
}

func (ts *TestifySuite) SetupTest() {
	ts.env = ts.NewTestWorkflowEnvironment()
}

func (ts *TestifySuite) Test_StartWorkflow() {
	ts.env.ExecuteWorkflow(workflow.StartWorkflow, time.Duration(time.Second))
	time.Sleep(time.Second * 2)

	ts.True(ts.env.IsWorkflowCompleted())
	ts.NoError(ts.env.GetWorkflowError())

	value, err := ts.env.QueryWorkflow("state")

	ts.Nil(err)

	var state workflow.State
	err = value.Get(&state)

	ts.Nil(err)
	ts.Equal(workflow.State_COMPLETED, state)
}
