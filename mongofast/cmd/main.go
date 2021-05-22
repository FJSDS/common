package main

import (
	"context"
	"fmt"
	"time"

	"github.com/FJSDS/common/mongofast"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

type Test0 struct {
	A int64
	B string
}

func (this_ *Test0) MarshalBSON() ([]byte, error) {
	b := make([]byte, 0, 256)
	b = mongofast.WriteHead(b)
	b = mongofast.WriteInt64(b, "a", this_.A)
	b = mongofast.WriteString(b, "b", this_.B)
	return mongofast.WriteEnd(b), nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017").SetMaxPoolSize(0))
	Must(err)
	defer client.Disconnect(context.Background())
	Must(client.Ping(context.Background(), readpref.Primary()))
	out, err := client.Database("test").Collection("test0").InsertOne(context.Background(), &Test0{A: 1212312313, B: "45123123126"})
	Must(err)
	fmt.Println(out.InsertedID)
}
