package Room

import "encoding/json"
import "Socket"
import "Radio"

//import "fmt"

func (m *Room) handleJoin(data []byte, client *Socket.SocketClient) {
	req := &JoinRoomRequest{}
	json.Unmarshal(data, &req)
	var resp = JoinRoomResponse{
		"login",
		true,
		JoinRoomInfo{
			m.Options.Name,
			m.radio.FileSize(),
			SizeInfo{
				m.Options.Width,
				m.Options.Height,
			},
			"asdfasdfuhwef",
		},
		0,
	}
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
	m.radio.SingleSendChan <- Radio.RadioSingleSendPart{
		Data:   data,
		Client: client,
	}
}
