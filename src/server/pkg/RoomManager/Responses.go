package RoomManager

//import "encoding/json"

type RoomPublicInfo struct {
	Name          string `json:"name"`
	CurrentLoad   int    `json:"currentload"`
	MaxLoad       int    `json:"maxload"`
	Private       bool   `json:"private"`
	ServerAddress string `json:"serveraddress"`
	Port          uint16 `json:"port"`
}

type RoomListResponse struct {
	Response string           `json:"response"`
	Result   bool             `json:"result"`
	RoomList []RoomPublicInfo `json:"roomlist"`
	ErrCode  int              `json:"errcode"`
}

type NewRoomInfoForReply struct {
	Port     uint16 `json:"port"`
	Key      string `json:"key"`
	Password string `json:"password"`
}

type NewRoomResponse struct {
	Response string              `json:"response"`
	Result   bool                `json:"result"`
	Info     NewRoomInfoForReply `json:"info"`
	ErrCode  int                 `json:"errcode"`
}
