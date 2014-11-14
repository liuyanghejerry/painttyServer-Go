package Router

import "encoding/json"
import "sync"
import "Socket"

type RouterHandler func([]byte, *Socket.SocketClient)

type Router struct {
	table  map[string]*RouterHandler
	locker sync.RWMutex
}

func MakeRouter() *Router {
	return &Router{
		make(map[string]*RouterHandler),
		sync.RWMutex{},
	}
}

func (r *Router) OnMessage(data []byte, client *Socket.SocketClient) error {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	var request = result["request"].(string)

	r.locker.RLock()
	if val, ok := r.table[request]; ok {
		//do something here
		(*val)(data, client)
	}
	r.locker.RUnlock()
	return nil
}

func (r *Router) Register(request string, handler RouterHandler) {
	r.locker.Lock()
	defer r.locker.Unlock()
	r.table[request] = &handler
}
