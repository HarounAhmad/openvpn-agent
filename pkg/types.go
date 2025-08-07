package pkg

type Client struct {
	CN             string `json:"cn"`
	RealIP         string `json:"real_ip"`
	VpnIP          string `json:"vpn_ip"`
	BytesIn        int64  `json:"bytes_in"`
	BytesOut       int64  `json:"bytes_out"`
	ConnectedSince string `json:"connected_since"`
}

type Command struct {
	Action string `json:"action"`
	CN     string `json:"cn"`
}

type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
