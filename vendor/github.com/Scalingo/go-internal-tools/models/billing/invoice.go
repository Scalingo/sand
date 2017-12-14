package billing

import "gopkg.in/mgo.v2/bson"

type Invoice struct {
	ID               bson.ObjectId `bson:"_id"`
	OwnerID          bson.ObjectId `bson:"owner_id"`
	State            string        `bson:"state"`
	InvoiceNumber    string        `bson:"invoice_number"`
	OctobatInvoiceId string        `bson:"octobat_invoice_id,omitempty"`
	PdfURL           string        `bson:"pdf_url,omitempty"`
}
