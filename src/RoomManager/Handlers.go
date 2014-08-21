package RoomManager

import "encoding/json"
import "Socket"
import "Room"

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

func (m *RoomManager) handleNewRoom(data []byte, client *Socket.SocketClient) {
	req := &NewRoomInfoForRequest{}
	json.Unmarshal(data, &req)

	var options = Room.RoomOption{
		MaxLoad:    8,
		Width:      1080,
		Height:     720,
		Name:       "Kiss",
		EmptyClose: true,
	}
	var room, err = Room.ServeRoom(options)
	if err != nil {
		panic(err)
	}
	m.roomsLocker.Lock()
	m.rooms[room.Options.Name] = &room
	m.roomsLocker.Unlock()
	room.Run()
	go func() {
		for {
			select {
			case <-room.GoingClose:
				room.GoingClose <- true
				m.roomsLocker.Lock()
				delete(m.rooms, room.Options.Name)
				m.roomsLocker.Unlock()
				return
			default:
				//
			}
		}
	}()

	var resp = RoomListResponse{
		"roomlist",
		true,
		[]RoomPublicInfo{},
		0,
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	_, err = client.SendManagerPack(raw)
	if err != nil {
		panic(err)
	}
}
