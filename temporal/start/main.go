package main

import (
	"context"
	"log"

	"go.temporal.io/sdk/client"
)

func main() {
	// Create the client object just once per process
	c, err := client.Dial(client.Options{})

	if err != nil {
		log.Fatalln("Unable to create Temporal client:", err)
	}

	defer c.Close()

	input := "World !!"

	options := client.StartWorkflowOptions{
		ID:        "pay-invoice-2",
		TaskQueue: "WORKFLOW_QUEUE",
	}

	we, err := c.ExecuteWorkflow(context.Background(), options, "Workflow", input)
	if err != nil {
		log.Fatalln("Unable to start the Workflow:", err)
	}

	log.Printf("WorkflowID: %s RunID: %s\n", we.GetID(), we.GetRunID())

	var result string

	err = we.Get(context.Background(), &result)

	if err != nil {
		log.Fatalln("Unable to get Workflow result:", err)
	}

	log.Println(result)

}
