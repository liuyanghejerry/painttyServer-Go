package Room

type ClearAllAction struct {
	Action    string `json:"action"`
	Signature string `json:"signature"`
}

type CloseActionInfo struct {
	Reason int64 `json:"reason"`
}

type CloseAction struct {
	Action string          `json:"action"`
	Info   CloseActionInfo `json:"info"`
}

type KickAction struct {
	Action string `json:"action"`
}

type NotifyAction struct {
	Action  string `json:"action"`
	Content string `json:"content"`
}

type WelcomeMsgType struct {
	Content string `json:"content"`
}
