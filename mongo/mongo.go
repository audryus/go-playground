package main

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI("mongodb://admin:VRuAd2Nvmp4ELHh5@localhost:27017/").SetServerAPIOptions(serverAPI)
	// Create a new client and connect to the server
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	db := client.Database("test")

	// Send a ping to confirm a successful connection
	if err := db.RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		panic(err)
	}
	col := db.Collection("tracks")
	filter := bson.D{{Key: "_id", Value: "url"}}

	result := new(Track)

	if err := col.FindOne(context.TODO(), filter).Decode(result); err != nil {
		panic(err)
	}
	fmt.Printf("Track %+v\n\n", result)

	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")
}

type Track struct {
	Url     string `json:"url" bson:"_id"`
	Status  string `json:"status" bson:"status"`
	Storage string `json:"storage" bson:"storage"`
}
