package RoomManager

import "encoding/json"
import "Socket"
import "Room"
import "fmt"
import "github.com/syndtr/goleveldb/leveldb/opt"

func (m *RoomManager) handleRoomList(data []byte, client *Socket.SocketClient) {
	req := &RoomListRequest{}
	json.Unmarshal(data, &req)
	fmt.Println(req.Request)
	roomlist := make([]RoomPublicInfo, 0, 100)
	for _, v := range m.rooms {
		room := RoomPublicInfo{
			Name:          v.Options.Name,
			CurrentLoad:   v.CurrentLoad(),
			Private:       false, //TODO
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
		client.GoingClose <- true
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
	}
	room, err := Room.ServeRoom(options)
	if err != nil {
		panic(err)
	}
	m.roomsLocker.Lock()
	m.rooms[room.Options.Name] = room
	m.roomsLocker.Unlock()
	room.Run()
	// insert to db
	info_to_insert := room.Dump()
	write_opt := &opt.WriteOptions{false}
	fmt.Println(room.Options.Name, string(info_to_insert))
	m.db.Put([]byte("room-"+room.Options.Name), info_to_insert, write_opt)
	go func() {
		_, _ = <-room.GoingClose
		m.roomsLocker.Lock()
		delete(m.rooms, room.Options.Name)
		m.roomsLocker.Unlock()
		m.db.Delete([]byte("room-"+room.Options.Name), write_opt)
		return
	}()

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
