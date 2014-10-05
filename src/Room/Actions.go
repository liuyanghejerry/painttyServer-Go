package Room

type ClearAllAction struct {
	Action    string `json:"action"`
	Signature string `json:"signature"`
}

type KickAction struct {
	Action string `json:"action"`
}
