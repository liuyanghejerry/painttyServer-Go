package Room

import "encoding/json"
import "Socket"

//import "Radio"

import "fmt"

func (m *Room) handleJoin(data []byte, client *Socket.SocketClient) {
	req := &JoinRoomRequest{}
	json.Unmarshal(data, &req)
	var resp = JoinRoomResponse{
		"login",
		true,
		JoinRoomInfo{
			Name:        m.Options.Name,
			HistorySize: m.radio.FileSize(),
			Size: SizeInfo{
				m.Options.Width,
				m.Options.Height,
			},
			ClientId: genClientId(),
		},
		0,
	}
	// TODO: due with failure login
	// TODO: set flag for success logined user
	var raw, err = json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	_, err = client.SendCommandPack(raw)
	if err != nil {
		//panic(err)
		client.GoingClose <- true
	}
}

func (m *Room) handleHeartbeat(data []byte, client *Socket.SocketClient) {
	req := &HeartbeatRequest{}
	json.Unmarshal(data, &req)
	var resp = HeartbeatResponse{
		Response:  "heartbeat",
		Timestamp: req.Timestamp,
	}

	var raw, err = json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	fmt.Println(req, resp)
	_, err = client.SendCommandPack(raw)
	if err != nil {
		//panic(err)
		client.GoingClose <- true
	}
}

func (m *Room) handleArchiveSign(data []byte, client *Socket.SocketClient) {
	req := &ArchiveSignRequest{}
	json.Unmarshal(data, &req)
	var resp = ArchiveSignResponse{
		Response:  "archivesign",
		Signature: m.radio.Signature(),
		Result:    true,
		Errcode:   0,
	}

	var raw, err = json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	fmt.Println(req, resp)
	_, err = client.SendCommandPack(raw)
	if err != nil {
		//panic(err)
		client.GoingClose <- true
	}
}

func (m *Room) handleArchive(data []byte, client *Socket.SocketClient) {
	req := &ArchiveRequest{}
	json.Unmarshal(data, &req)

	var resp = ArchiveResponse{
		Response: "archive",
		Result:   false,
		Errcode:  900,
	}

	var startPos = req.Start
	var realLength = m.radio.FileSize()
	fmt.Println("real length", realLength)
	var dataLength = req.DataLength

	if dataLength != 0 {
		if startPos+dataLength <= realLength {
			dataLength = realLength - startPos
		}
	} else {
		dataLength = realLength
	}

	if startPos > realLength {
		resp.Errcode = 901
	}

	resp = ArchiveResponse{
		Response:   "archive",
		Signature:  m.radio.Signature(),
		DataLength: dataLength,
		Result:     true,
		Errcode:    0,
	}

	var raw, err = json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	fmt.Println(req, resp)
	_, err = client.SendCommandPack(raw)
	if err != nil {
		//panic(err)
		client.GoingClose <- true
	}

	m.radio.AddClient(client, startPos, dataLength)
}

func (m *Room) handleClearAll(data []byte, client *Socket.SocketClient) {
	req := &ClearAllRequest{}
	json.Unmarshal(data, &req)

	var resp = ClearAllResponse{
		Response: "clearall",
		Result:   false,
	}

	m.radio.Prune()

	var action = ClearAllAction{
		Action:    "clearall",
		Signature: m.radio.Signature(),
	}

	_ = action

	// TODO: broadcast to all

	resp.Result = true

	var raw, err = json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	fmt.Println(req, resp)
	_, err = client.SendCommandPack(raw)
	if err != nil {
		//panic(err)
		client.GoingClose <- true
	}
}
