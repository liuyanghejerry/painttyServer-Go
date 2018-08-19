package Hub

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestNormalSubPub(t *testing.T) {
	hub := Hub{}
	handler := Handler{
		Name: "hello",
		Callback: func(content interface{}) {
			value, ok := content.(string)
			if !ok || value != "test" {
				t.Fatal("Callback content error", ok, content)
			}
		},
	}
	hub.Sub("my_topic", handler)
	hub.Pub("my_topic", "test")
}

func TestMultiplePub(t *testing.T) {
	hub := Hub{}
	counter := 0
	handler := Handler{
		Name: "hello",
		Callback: func(content interface{}) {
			value, ok := content.(string)
			if !ok || value != "test" {
				t.Fatal("Callback content error", ok, content)
			}
			counter++
		},
	}
	hub.Sub("my_topic", handler)
	hub.Pub("my_topic", "test")
	hub.Pub("my_topic", "test")
	hub.Pub("my_topic", "test")
	if counter != 3 {
		t.Fatal("Multiple Pub failed", counter)
	}
}

func TestUnmatchedPub(t *testing.T) {
	hub := Hub{}
	counter := 0
	handler := Handler{
		Name: "hello",
		Callback: func(content interface{}) {
			value, ok := content.(string)
			if !ok || value != "test" {
				t.Fatal("Callback content error", ok, content)
			}
			counter++
		},
	}
	hub.Sub("my_topic", handler)
	hub.Pub("not_my_topic", "test")
	if counter != 0 {
		t.Fatal("Unmatched topic should not be involved", counter)
	}
}

func TestUnSub(t *testing.T) {
	hub := Hub{}
	counter := 0
	handler := Handler{
		Name: "hello",
		Callback: func(content interface{}) {
			value, ok := content.(string)
			if !ok || value != "test" {
				t.Fatal("Callback content error", ok, content)
			}
			counter++
		},
	}
	hub.Sub("my_topic", handler)
	hub.Pub("my_topic", "test")
	hub.Pub("my_topic", "test")
	hub.Unsub("my_topic", handler)
	hub.Pub("my_topic", "test")
	if counter != 2 {
		t.Fatal("Unsub failed", counter)
	}
}

func TestConcurrent(t *testing.T) {
	hub := Hub{}
	var counter int64
	counter = 0
	var wg sync.WaitGroup
	handler := Handler{
		Name: "hello",
		Callback: func(content interface{}) {
			defer wg.Done()
			t.Log("callback invoked")
			value, ok := content.(string)
			if !ok || value != "test" {
				t.Fatal("Callback content error", ok, content)
			}
			atomic.AddInt64(&counter, 1)
		},
	}
	wg.Add(4)
	hub.Sub("my_topic", handler)
	go hub.Pub("my_topic", "test")
	go hub.Pub("my_topic", "test")
	go hub.Pub("my_topic", "test")
	go hub.Pub("my_topic", "test")
	wg.Wait()
	hub.Unsub("my_topic", handler)
	go hub.Pub("my_topic", "test")
	if counter != 4 {
		t.Fatal("Unsub failed", counter)
	}
}
