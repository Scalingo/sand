package httpresp

type Error struct {
	Error_ string `json:"error"`
}

func (e Error) Error() string {
	return e.Error_
}
