package RoomManager

import "encoding/json"
import "Socket"
import "Room"
import "github.com/syndtr/goleveldb/leveldb/opt"

func (m *RoomManager) handleRoomList(data []byte, client *Socket.SocketClient) {
	req := &RoomListRequest{}
	json.Unmarshal(data, &req)
	debugOut(req.Request)
	m.roomsLocker.Lock()
	var roomsCopy = map[string]*Room.Room{}
	for k, v := range m.rooms {
		roomsCopy[k] = v
	}
	m.roomsLocker.Unlock()
	debugOut("room count", len(roomsCopy))
	roomlist := make([]RoomPublicInfo, 0, len(roomsCopy))
	for _, v := range roomsCopy {
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

	if code := m.limitRoomOption(&options); code != 0 {
		var resp = NewRoomResponse{
			Response: "newroom",
			Result:   false,
			ErrCode:  code,
		}

		raw, err := json.Marshal(resp)
		if err != nil {
			panic(err)
		}
		_, err = client.SendManagerPack(raw)
		if err != nil {
			panic(err)
		}
		return
	}

	room, err := Room.ServeRoom(options)
	if err != nil {
		panic(err)
	}
	m.roomsLocker.Lock()
	m.rooms[room.Options.Name] = room
	m.roomsLocker.Unlock()
	go func(room *Room.Room, m *RoomManager) {
		roomName := room.Options.Name
		room.Run()
		m.waitRoomClosed(roomName)
	}(room, m)

	// insert to db
	info_to_insert := room.Dump()
	write_opt := &opt.WriteOptions{false}
	debugOut(room.Options.Name, string(info_to_insert))
	m.db.Put([]byte("room-"+room.Options.Name), info_to_insert, write_opt)

	var resp = NewRoomResponse{
		Response: "newroom",
		Result:   true,
		Info: NewRoomInfoForReply{
			Port:     room.Port(),
			Key:      room.Key(),
			Password: room.Password(),
		},
		ErrCode: 0,
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
