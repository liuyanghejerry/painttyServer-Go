package RoomManager

import "Room"
import "encoding/json"

func parseRoomRuntimeInfo(data []byte) *Room.RoomRuntimeInfo {
	info := &Room.RoomRuntimeInfo{}
	json.Unmarshal(data, info)
	return info
}
