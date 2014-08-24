package Room

type JoinRoomRequest struct {
	Request  string `json: "request"`
	Password string `json: "password"`
	Name     string `json: "name"`
}
