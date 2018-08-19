package Hub

import (
	"log"
	"sync"
)

const BACKLOG_LEN = 100

type Handler struct {
	Name     string
	Callback func(content interface{})
}

type TopicQueue struct {
	topic     string
	queueLock sync.Mutex
	handlers  []Handler
}

func (t *TopicQueue) AddHandler(handler Handler) {
	defer t.queueLock.Unlock()
	t.queueLock.Lock()
	t.handlers = append(t.handlers, handler)
}

func (t *TopicQueue) RemoveHandler(handlerName string) {
	defer t.queueLock.Unlock()
	t.queueLock.Lock()
	index := -1
	for i := 0; i < len(t.handlers); i++ {
		if t.handlers[i].Name == handlerName {
			index = i
			break
		}
	}
	if index == -1 {
		return
	}
	t.handlers = append(t.handlers[:index], t.handlers[index+1:]...)
}

func (t *TopicQueue) InvokeHandler(content interface{}) {
	defer t.queueLock.Unlock()
	t.queueLock.Lock()
	for i := 0; i < len(t.handlers); i++ {
		t.handlers[i].Callback(content)
	}
}

type Hub struct {
	topicCollection sync.Map
	hubLock         sync.Mutex
}

func (h *Hub) Sub(topic string, handler Handler) {
	h.hubLock.Lock()
	defer h.hubLock.Unlock()
    handlers := []Handler{handler}
    queue := &TopicQueue{
		topic:    topic,
		handlers: handlers,
	}
	oldTopicQueueStub, loaded := h.topicCollection.LoadOrStore(topic, queue)
	if !loaded {
		return
	}
	queue, ok := oldTopicQueueStub.(*TopicQueue)
	if !ok {
		// should never reach here
		log.Println("oldTopicQueueStub type assertion failed", topic)
		return
	}
	queue.AddHandler(handler)
	h.topicCollection.Store(topic, queue)
}

func (h *Hub) Unsub(topic string, handler Handler) {
	h.hubLock.Lock()
	defer h.hubLock.Unlock()
    handlers := []Handler{handler}
    queue := &TopicQueue{
		topic:    topic,
		handlers: handlers,
	}
	oldTopicQueueStub, loaded := h.topicCollection.LoadOrStore(topic, queue)
	if !loaded {
		return
	}
	queue, ok := oldTopicQueueStub.(*TopicQueue)
	if !ok {
		// should never reach here
		log.Println("oldTopicQueueStub type assertion failed", topic)
		return
	}
	log.Println(2)
	queue.RemoveHandler(handler.Name)
	h.topicCollection.Store(topic, queue)
}

func (h *Hub) Pub(topic string, content interface{}) {
	h.hubLock.Lock()
	defer h.hubLock.Unlock()
    handlers := []Handler{}
    queue := &TopicQueue{
		topic:    topic,
		handlers: handlers,
	}
	stub, loaded := h.topicCollection.LoadOrStore(topic, queue)
	if !loaded {
    	queue.InvokeHandler(content)
		return
	}
	queue, ok := stub.(*TopicQueue)
	if !ok {
		log.Println("TopicQueue type assertion failed", topic, stub)
		return
	}
	queue.InvokeHandler(content)
}

func (h *Hub) Clear() {
	h.hubLock.Lock()
	defer h.hubLock.Unlock()
	h.topicCollection = sync.Map{}
}

func MakeHub() Hub {
    return Hub{
        topicCollection: sync.Map{},
        hubLock: sync.Mutex{},
    }
}
