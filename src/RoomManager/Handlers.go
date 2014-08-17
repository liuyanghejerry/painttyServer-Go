package RoomManager

import "encoding/json"
import "Socket"

func (m *RoomManager) handleRoomList(data []byte, client *Socket.SocketClient) {
	req := &RoomListRequest{}
	json.Unmarshal(data, &req)
	var resp = RoomListResponse{
		"roomlist",
		true,
		[]RoomPublicInfo{},
		0,
	}
	var raw, err = json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	_, err = client.SendManagerPack(raw)
	if err != nil {
		panic(err)
	}
}
