package billing

import "gopkg.in/mgo.v2/bson"

type Profile struct {
	ID                bson.ObjectId `bson:"_id"`
	OwnerID           bson.ObjectId `bson:"owner_id"`
	OctobatCustomerId string        `bson:"octobat_customer_id,omitempty"`
}
