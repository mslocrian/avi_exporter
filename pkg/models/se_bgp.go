package models

type SeBGP struct {
	Active  int         `json:"active"`
	LocalAS int         `json:"local_as"`
	Name    string      `json:"name"`
	PeerBMP []int       `json:"peer_bmp"`
	Peers   []SeBGPPeer `json:"peers"`
	ProcID  string      `json:"peer_id"`
	Routes  []string    `json:"routes"`
	SeUUID  string      `json:"se_uuid"`
	VRF     int         `json:"vrf"`
	VSNames []string    `json:"vs_names"`
}

type SeBGPPeer struct {
	Active          int    `json:"active"`
	AdvertiseSnatIP bool   `json:"advertise_snat_ip"`
	AdvertiseVIP    bool   `json:"advertise_vip"`
	BFD             bool   `json:"bfd"`
	PeerID          int    `json:"peer_id"`
	PeerIP          string `json:"peer_ip"`
	PeerState       string `json:"peer_state"`
	RemoteAS        int    `json:"remote_as"`
}
