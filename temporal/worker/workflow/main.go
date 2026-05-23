package main

import (
	"log"
	app "myplayground/temporal"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("Unable to create Temporal client.", err)
	}
	defer c.Close()

	w := worker.New(c, "WORKFLOW_QUEUE", worker.Options{})

	// This worker hosts both Workflow and Activity functions.
	w.RegisterWorkflow(app.Workflow)
	//w.RegisterActivity(app.Activity)

	// Start listening to the Task Queue.
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start Worker", err)
	}
}
