package Router

import "encoding/json"
import "Socket"

type RouterHandler func([]byte, *Socket.SocketClient)

type Router struct {
	table map[string]RouterHandler
}

func MakeRouter() *Router {
	return &Router{
		make(map[string]RouterHandler),
	}
}

func (r *Router) OnMessage(data []byte, client *Socket.SocketClient) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		panic(err)
	}

	var request = result["request"].(string)

	if val, ok := r.table[request]; ok {
		//do something here
		val(data, client)
	}
}

func (r *Router) Register(request string, handler RouterHandler) {
	r.table[request] = handler
}
