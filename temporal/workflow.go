package app

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func Workflow(ctx workflow.Context, input string) (string, error) {
	fmt.Println("Workflow executed ...")
	// RetryPolicy specifies how to automatically handle retries if an Activity fails.
	retrypolicy := &temporal.RetryPolicy{
		InitialInterval:        time.Second,
		BackoffCoefficient:     2.0,
		MaximumInterval:        5 * time.Second,
		MaximumAttempts:        2, // 0 is unlimited retries
		NonRetryableErrorTypes: []string{"InvalidAccountError", "InsufficientFundsError"},
	}

	options := workflow.ActivityOptions{
		// Timeout options specify when to automatically timeout Activity functions.
		StartToCloseTimeout: time.Minute,
		// Optionally provide a customized RetryPolicy.
		// Temporal retries failed Activities by default.
		RetryPolicy: retrypolicy,
		TaskQueue:   "ACTIVITY_QUEUE",
	}

	// Apply the options.
	ctx = workflow.WithActivityOptions(ctx, options)

	var output string
	withdrawErr := workflow.ExecuteActivity(ctx, Activity, input).Get(ctx, &output)
	fmt.Println("activity returned  ...")
	if withdrawErr != nil {
		//return "", withdrawErr
	}

	return "output", nil
}
