package Room

import (
	"ErrorCode"
	"Socket"
	"encoding/json"
	"log"
)

func sendToClient(resp interface{}, client *Socket.SocketClient) {
	var raw, err = json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	_, err = client.SendCommandPack(raw)
	if err != nil {
		client.Close()
	}
}

func (m *Room) handleJoin(data []byte, client *Socket.SocketClient) {
	req := &JoinRoomRequest{}
	json.Unmarshal(data, &req)

	var resp = JoinRoomResponse{
		"login",
		false,
		JoinRoomInfo{},
		ErrorCode.LOGIN_UNKOWN,
	}

	if req.Password != m.Options.Password {
		resp.ErrCode = ErrorCode.LOGIN_PWD_INCORRECT
		sendToClient(resp, client)
		return
	}

	if m.CurrentLoad() > m.Options.MaxLoad {
		resp.ErrCode = ErrorCode.LOGIN_ROOM_IS_FULL
		sendToClient(resp, client)
		return
	}

	clientId := m.genClientId()

	resp = JoinRoomResponse{
		"login",
		true,
		JoinRoomInfo{
			Name:        m.Options.Name,
			HistorySize: m.radio.FileSize(),
			Size: SizeInfo{
				m.Options.Width,
				m.Options.Height,
			},
			ClientId: clientId,
		},
		0,
	}
	sendToClient(resp, client)

	m.locker.Lock()
	defer m.locker.Unlock()
	if user, ok := m.clients[client]; ok {
		user.clientId = clientId
	} else {
		panic("handleJoin found unclean client")
	}
}

func (m *Room) handleHeartbeat(data []byte, client *Socket.SocketClient) {
	req := &HeartbeatRequest{}
	json.Unmarshal(data, &req)
	var resp = HeartbeatResponse{
		Response:  "heartbeat",
		Timestamp: req.Timestamp,
	}

	sendToClient(resp, client)
}

func (m *Room) handleArchiveSign(data []byte, client *Socket.SocketClient) {
	if !m.hasUser(client) {
		return
	}

	req := &ArchiveSignRequest{}
	json.Unmarshal(data, &req)
	var resp = ArchiveSignResponse{
		Response:  "archivesign",
		Signature: m.radio.Signature(),
		Result:    true,
		Errcode:   0,
	}

	sendToClient(resp, client)
	log.Println(req, resp)
}

func (m *Room) handleArchive(data []byte, client *Socket.SocketClient) {
	if !m.hasUser(client) {
		return
	}
	req := &ArchiveRequest{}
	json.Unmarshal(data, &req)

	var resp = ArchiveResponse{
		Response: "archive",
		Result:   false,
		Errcode:  900,
	}

	var startPos = req.Start
	var realLength = m.radio.FileSize()
	var dataLength = req.DataLength

	if startPos > realLength {
		startPos = realLength
		dataLength = 0
	}

	if startPos+dataLength >= realLength {
		dataLength = realLength - startPos
	}

	resp = ArchiveResponse{
		Response:   "archive",
		Signature:  m.radio.Signature(),
		DataLength: dataLength,
		Result:     true,
		Errcode:    0,
	}

	log.Println(req, resp)
	sendToClient(resp, client)

	if resp.Result {
		log.Println("startPos", startPos, "dataLength", dataLength)
		go m.radio.AddClient(client, startPos, dataLength)
	}
}

func (m *Room) handleClearAll(data []byte, client *Socket.SocketClient) {
	if !m.hasUser(client) {
		return
	}
	req := &ClearAllRequest{}
	json.Unmarshal(data, &req)

	var resp = ClearAllResponse{
		Response: "clearall",
		Result:   false,
	}

	if req.Key != m.Key() {
		sendToClient(resp, client)
		return
	}

	m.archiveSign = m.radio.Prune()

	var action = ClearAllAction{
		Action:    "clearall",
		Signature: m.radio.Signature(),
	}

	resp.Result = true
	log.Println(req, resp)
	sendToClient(resp, client)
	m.broadcastCommand(action)
}
