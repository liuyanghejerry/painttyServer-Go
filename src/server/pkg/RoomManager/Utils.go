package RoomManager

import (
	"encoding/json"
	"server/pkg/Config"
	"server/pkg/ErrorCode"
	"server/pkg/Room"
	"sync/atomic"
)

func parseRoomRuntimeInfo(data []byte) *Room.RoomRuntimeInfo {
	info := &Room.RoomRuntimeInfo{}
	json.Unmarshal(data, info)
	return info
}

func (m *RoomManager) limitRoomOption(option *Room.RoomOption) int {
	maxLoad := Config.ReadConfInt("max_load", 8)
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

	maxRoomCount := Config.ReadConfInt("max_room_count", 1000)
	if int(atomic.LoadInt32(&m.currentRoomCount)) >= maxRoomCount {
		return ErrorCode.NEW_ROOM_TOO_MANY_ROOMS
	}
	return 0
}
