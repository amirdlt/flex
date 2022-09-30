package main

import (
	"fmt"
	"github.com/amirdlt/ffvm"
	. "github.com/amirdlt/flex"
	. "github.com/amirdlt/flex/util"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type user struct {
	Name        string `json:"name" bson:"name" ffvm:",min_len=10"`
	Age         int    `json:"age" bson:"age"`
	PhoneNumber string `json:"phoneNumber" bson:"phoneNumber"`
	Id          string `json:"id" bson:"id"`
}

type product struct {
	Name   string  `json:"name" bson:"name"`
	Id     string  `json:"id" bson:"id"`
	Price  float64 `json:"price" bson:"price"`
	Weight float64 `json:"weight" bson:"weight"`
}

type in struct {
	user user
	*BasicInjector
}

type history struct {
	UserId           string `json:"userId" bson:"userId"`
	Api              string `json:"api" bson:"api"`
	ResultStatusCode int    `json:"resultStatusCode" bson:"resultStatusCode"`
}

func simpleServer() {
	s := New(M{}, func(i *BasicInjector) *in {
		return &in{
			BasicInjector: i,
		}
	})

	s.SetDefaultMongoClient("mongodb://localhost:27017")
	db := s.GetDefaultMongoClient().GetDatabase("simple_app")

	s.Group("/user").POST("/create", func(i *in) Result {
		newUser := i.RequestBody().(user)

		t := time.Now()
		fmt.Println(ffvm.Validate(&newUser))
		fmt.Println(time.Since(t).Nanoseconds())

		if newUser.Name == "" {
			return i.WrapBadRequestErr("name must be provided")
		}

		if newUser.Age < 1 {
			return i.WrapBadRequestErr("age must be greater than zero")
		}

		if index, err := db.GetCollection("user").CountDocuments(i.Context(), bson.M{}); err != nil {
			return i.WrapInternalErr("internal, err=" + err.Error())
		} else {
			newUser.Id = fmt.Sprint(index)
		}

		if _, err := db.GetCollection("user").InsertOne(i.Context(), newUser); err != nil {
			return i.WrapInternalErr("internal, err=" + err.Error())
		}

		return i.WrapOk(M{
			"status": "user created successfully, id=" + newUser.Id,
		})
	}, user{})

	s.Group("/user").GET("/:userId", func(i *in) Result {
		var user user
		if err := db.GetCollection("user").FindOne(i.Context(),
			bson.M{"id": i.PathParameter("userId")}).Decode(&user); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return i.WrapBadRequestErr("no user found")
			}

			return i.WrapBadRequestErr("internal, err=" + err.Error())
		}

		return i.WrapOk(user)
	}, NoBody{})

	_ = s.Run()
}
