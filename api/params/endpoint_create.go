package params

type EndpointCreate struct {
	NetworkID      string           `json:"network_id"`
	Activate       bool             `json:"activate"`
	ActivateParams EndpointActivate `json:"activate_params"`
	IPv4Address    string           `json:"ipv4_address"`
	MacAddress     string           `json:"mac_address"`
}
