package workflow

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

type State int32

const (
	State_STARTED   State = 0
	State_COMPLETED State = 1
	State_FAILED    State = 2
)

var State_name = map[State]string{
	0: "started",
	1: "completed",
	2: "failed",
}

func init() {
	workflow.Register(StartWorkflow)
	activity.Register(sleep)
}

func StartWorkflow(ctx workflow.Context, waitingTime time.Duration) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("workflow start")

	currentState := State_STARTED
	err := workflow.SetQueryHandler(ctx, "state", func() (State, error) {
		return currentState, nil
	})
	if err != nil {
		currentState = State_FAILED
		return err
	}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    waitingTime + time.Duration(time.Second),
		HeartbeatTimeout:       0,
	})

	logger.Info("execute sleep activity")

	var result bool
	err = workflow.ExecuteActivity(ctx, sleep, waitingTime).Get(ctx, &result)
	if err != nil {
		currentState = State_FAILED

		logger.Error("failed to execute workflow", zap.Error(err))

		return err
	}

	currentState = State_COMPLETED

	logger.Info("workflow complete")

	return nil
}

func sleep(ctx context.Context, waitingTime time.Duration) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(fmt.Sprintf("waiting time: %s", waitingTime))

	time.Sleep(waitingTime)

	return true, nil
}
