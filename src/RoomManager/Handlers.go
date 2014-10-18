package RoomManager

import "encoding/json"
import "Socket"
import "Room"
import "log"
import "github.com/syndtr/goleveldb/leveldb/opt"

func (m *RoomManager) handleRoomList(data []byte, client *Socket.SocketClient) {
	req := &RoomListRequest{}
	json.Unmarshal(data, &req)
	log.Println(req.Request)
	roomlist := make([]RoomPublicInfo, 0, 100)
	m.roomsLocker.Lock()
	defer m.roomsLocker.Unlock()
	log.Println("room count", len(m.rooms))
	for _, v := range m.rooms {
		room := RoomPublicInfo{
			Name:          v.Options.Name,
			CurrentLoad:   v.CurrentLoad(),
			Private:       len(v.Options.Password) > 0,
			MaxLoad:       v.Options.MaxLoad,
			ServerAddress: "0.0.0.0",
			Port:          v.Port(),
		}
		roomlist = append(roomlist, room)
	}
	var resp = RoomListResponse{
		"roomlist",
		true,
		roomlist,
		0,
	}
	var raw, err = json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	_, err = client.SendManagerPack(raw)
	if err != nil {
		//panic(err)
		//client.GoingClose <- true
		client.Close()
	}
}

func (m *RoomManager) handleNewRoom(data []byte, client *Socket.SocketClient) {
	req := &NewRoomRequest{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		panic(err)
	}

	var options = Room.RoomOption{
		MaxLoad:    req.Info.MaxLoad,
		Width:      req.Info.Size.Width,
		Height:     req.Info.Size.Height,
		Name:       req.Info.Name,
		EmptyClose: req.Info.EmptyClose,
		WelcomeMsg: req.Info.WelcomeMsg,
		Password:   req.Info.Password,
	}
	room, err := Room.ServeRoom(options)
	if err != nil {
		panic(err)
	}
	m.roomsLocker.Lock()
	m.rooms[room.Options.Name] = room
	m.roomsLocker.Unlock()
	go func(room *Room.Room) {
		roomName := room.Options.Name
		room.Run()
		m.waitRoomClosed(roomName)
	}(room)

	// insert to db
	info_to_insert := room.Dump()
	write_opt := &opt.WriteOptions{false}
	log.Println(room.Options.Name, string(info_to_insert))
	m.db.Put([]byte("room-"+room.Options.Name), info_to_insert, write_opt)

	var resp = NewRoomResponse{
		"newroom",
		true,
		NewRoomInfoForReply{
			room.Port(),
			room.Key(),
			"",
		},
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
