package models

import "gopkg.in/mgo.v2/bson"

type User struct {
	ID    bson.ObjectId `bson:"_id"`
	Email string        `bson:"email"`
}
