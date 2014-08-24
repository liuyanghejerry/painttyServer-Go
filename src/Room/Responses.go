package Room

type SizeInfo struct {
	Width  int `json:"width"`
	Height int `json:"height"`
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
	ErrCode  int          `json:"errcode"`
}
