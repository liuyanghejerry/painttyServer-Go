package Room

type JoinRoomRequest struct {
	Request  string `json: "request"`
	Password string `json: "password"`
	Name     string `json: "name"`
}

type HeartbeatRequest struct {
	Request   string `json: "request"`
	Timestamp int64  `json: "timestamp"`
}

type ArchiveSignRequest struct {
	Request string `json: "request"`
}

type ArchiveRequest struct {
	Request    string `json: "request"`
	Start      int64  `json:"start"`
	DataLength int64  `json:"datalength"`
}

type ClearAllRequest struct {
	Request string `json: "request"`
	Key     string `json:"key"`
}

type KickRequest struct {
	Request  string `json: "request"`
	Key      string `json: "key"`
	ClientId string `json: "clientid"`
}

type CloseRequest struct {
	Request string `json: "request"`
	Key     string `json: "key"`
}

type CheckoutRequest struct {
	Request string `json: "request"`
	Key     string `json: "key"`
}

type OnlineListRequest struct {
	Request  string `json: "request"`
	ClientId string `json: "clientid"`
}
