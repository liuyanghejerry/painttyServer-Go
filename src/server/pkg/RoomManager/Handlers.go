package RoomManager

import "log"
import "sync/atomic"
import "encoding/json"
import "server/pkg/Socket"
import "server/pkg/Room"
import "github.com/syndtr/goleveldb/leveldb/opt"

func (m *RoomManager) handleRoomList(data []byte, client *Socket.SocketClient) {
	req := &RoomListRequest{}
	json.Unmarshal(data, &req)
	debugOut(req.Request)
	roomlist := make([]RoomPublicInfo, 0, atomic.LoadUint32(&m.currentRoomCount))
    m.rooms.Range(func (key, value interface{}) bool {
        roomInstance, ok := value.(*Room.Room)
        if !ok {
            log.Panicln("Read rooms from RoomManager failed: instance convertion failed")
        }
        room := RoomPublicInfo{
			Name:          roomInstance.Options.Name,
			CurrentLoad:   roomInstance.CurrentLoad(),
			Private:       len(roomInstance.Options.Password) > 0,
			MaxLoad:       roomInstance.Options.MaxLoad,
			ServerAddress: "0.0.0.0",
			Port:          roomInstance.Port(),
		}
        roomlist = append(roomlist, room)
        return true
    })
	var resp = RoomListResponse{
		"roomlist",
		true,
		roomlist,
		0,
	}
	var raw, err = json.Marshal(resp)
	if err != nil {
		log.Panicln(err)
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
    m.rooms.Store(room.Options.Name, room)
    atomic.AddUint32(&m.currentRoomCount, 1)
    go func(room *Room.Room, m *RoomManager) {
		roomName := room.Options.Name
		room.Run()
		m.waitRoomClosed(roomName)
	}(room, m)

	// insert to db
	info_to_insert := room.Dump()
	write_opt := &opt.WriteOptions{}
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
