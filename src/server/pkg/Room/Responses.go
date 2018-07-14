package Room

type SizeInfo struct {
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

type JoinRoomInfo struct {
	Name        string   `json:"name"`
	HistorySize int64    `json:"historysize"`
	Size        SizeInfo `json:"size"`
	ClientId    string   `json:"clientid"`
}

type JoinRoomResponse struct {
	Response string       `json:"response"`
	Result   bool         `json:"result"`
	RoomList JoinRoomInfo `json:"info"`
	ErrCode  int64        `json:"errcode"`
}

type HeartbeatResponse struct {
	Response  string `json:"response"`
	Timestamp int64  `json:"timestamp"`
}

type ArchiveSignResponse struct {
	Response  string `json:"response"`
	Result    bool   `json:"result"`
	Signature string `json:"signature"`
	Errcode   int64  `json:"errcode"`
}

type ArchiveResponse struct {
	Response   string `json:"response"`
	Result     bool   `json:"result"`
	Signature  string `json:"signature"`
	DataLength int64  `json:"datalength"`
	Errcode    int64  `json:"errcode"`
}

type ClearAllResponse struct {
	Response string `json:"response"`
	Result   bool   `json:"result"`
}

type KickResponse struct {
	Response string `json:"response"`
	Result   bool   `json:"result"`
}

type CloseResponse struct {
	Response string `json:"response"`
	Result   bool   `json:"result"`
}

type CheckoutResponse struct {
	Response string `json:"response"`
	Result   bool   `json:"result"`
	Cycle    int64  `json:"cycle"`
	Errcode  int64  `json:"errcode"`
}

type OnlineListItem struct {
	Name     string `json:"name"`
	ClientId string `json:"clientid"`
}

type OnlineListResponse struct {
	Response   string           `json:"response"`
	Result     bool             `json:"result"`
	OnlineList []OnlineListItem `json:"onlinelist"`
}
