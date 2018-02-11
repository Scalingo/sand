package params

type EndpointActivate struct {
	NSHandlePath string `json:"ns_handle_path"`
	SetAddr      bool   `json:"set_addr"`
	MoveVeth     bool   `json:"move_veth"`
}
