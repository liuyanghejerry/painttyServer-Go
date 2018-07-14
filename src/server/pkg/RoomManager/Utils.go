package RoomManager

import (
    "encoding/json"
    "sync/atomic"
    "server/pkg/Room"
    "server/pkg/ErrorCode"
    "server/pkg/Config"
)

func parseRoomRuntimeInfo(data []byte) *Room.RoomRuntimeInfo {
	info := &Room.RoomRuntimeInfo{}
	json.Unmarshal(data, info)
	return info
}

func (m *RoomManager) limitRoomOption(option *Room.RoomOption) int {
	maxLoad := Config.GetConfig()["max_load"].(int)
	if option.MaxLoad > maxLoad || option.MaxLoad < 1 {
		return ErrorCode.NEW_ROOM_INVALID_MAXLOAD
	}

	canvasSize := option.Width * option.Height

	if canvasSize > 6998400 || canvasSize <= 0 {
		return ErrorCode.NEW_ROOM_INVALID_CANVAS
	}

	nameLen := len(option.Name)
	if nameLen > 20 || nameLen < 0 {
		return ErrorCode.NEW_ROOM_INVALID_NAME
	}

	if _, ok := m.rooms.Load(option.Name); ok {
		return ErrorCode.NEW_ROOM_NAME_COLLISSION
	}

	welcomeMsgLen := len(option.WelcomeMsg)
	if welcomeMsgLen > 40 {
		return ErrorCode.NEW_ROOM_INVALID_WELCMSG
	}

	pwdLen := len(option.Password)
	if pwdLen > 16 {
		return ErrorCode.NEW_ROOM_INVALID_PWD
	}

	maxRoomCount := Config.GetConfig()["max_room_count"].(int)
	if int(atomic.LoadUint32(&m.currentRoomCount)) >= maxRoomCount {
		return ErrorCode.NEW_ROOM_TOO_MANY_ROOMS
	}
	return 0
}
