package RoomManager

import "Room"
import "encoding/json"
import "ErrorCode"
import "Config"

func parseRoomRuntimeInfo(data []byte) (*Room.RoomRuntimeInfo, error) {
	info := &Room.RoomRuntimeInfo{}
	err := json.Unmarshal(data, info)
	return info, err
}

func (m *RoomManager) limitRoomOption(option *Room.RoomOption) int {
	config := Config.GetConfig()

	maxLoad := config["maxLoad"].(int)
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

	m.roomsLocker.Lock()
	defer m.roomsLocker.Unlock()
	if _, ok := m.rooms[option.Name]; ok {
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

	if len(m.rooms) >= 500 {
		return ErrorCode.NEW_ROOM_TOO_MANY_ROOMS
	}
	return 0
}
