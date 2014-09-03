package RoomManager

//import "encoding/json"

type RoomListRequest struct {
	Request string `json:"request"`
}

type NewRoomSize struct {
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

type NewRoomInfoForRequest struct {
	Name       string      `json:"name"`
	MaxLoad    int         `json:"maxload"`
	WelcomeMsg string      `json:"welcomemsg"`
	EmptyClose bool        `json:"emptyclose"`
	Size       NewRoomSize `json:"size"`
	Password   string      `json:"password"`
}

type NewRoomRequest struct {
	Request string                `json:"request"`
	Info    NewRoomInfoForRequest `json:"info"`
}
