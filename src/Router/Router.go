package Router

import "encoding/json"
import "sync"
import "Socket"

type RouterHandler func([]byte, *Socket.SocketClient)

type Router struct {
	key    string
	table  map[string]*RouterHandler
	locker sync.RWMutex
}

func MakeRouter(key string) *Router {
	return &Router{
		key,
		make(map[string]*RouterHandler),
		sync.RWMutex{},
	}
}

func (r *Router) OnMessage(data []byte, client *Socket.SocketClient) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	var result map[string]interface{}
	if err = json.Unmarshal(data, &result); err != nil {
		return err
	}

	var request = result[r.key].(string)

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
