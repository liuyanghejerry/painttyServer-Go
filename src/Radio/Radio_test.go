package Radio

import "testing"

import "sync"
import "log"

func TestRadioTaskList(t *testing.T) {
	var taskList = RadioTaskList{
		make([]RadioChunk, 0, 100),
		sync.Mutex{},
	}

	if taskList.Length() != 0 {
		t.Log("taskList size is incorrect")
		t.Error(taskList.Length())
	}

	taskList.Append([]RadioChunk{FileChunk{
		0,
		3600,
	}})

	if taskList.Length() != 1 {
		t.Log("taskList size is incorrect")
		t.Error(taskList.Length())
	}

	taskList.PushBack([]RadioChunk{FileChunk{
		3600,
		20,
	}})

	var item = taskList.PopBack().(FileChunk)

	if item.Start != 3600 {
		t.Log("taskList PopBack is incorrect")
		t.Error(item.Start)
	}

	if taskList.Length() != 1 {
		t.Log("taskList PopBack is incorrect")
		t.Error(taskList.Length())
	}

	item = taskList.PopFront().(FileChunk)
	item0 := item
	item1 := item
	item2 := item
	item3 := item
	item0.Start = 0
	item1.Start = 1
	item2.Start = 2
	item3.Start = 3
	taskList.Append([]RadioChunk{item0, item1, item2, item3})

	item = taskList.PopFront().(FileChunk)

	if item.Start != item0.Start {
		t.Log("taskList PopFront is incorrect")
		t.Error(item, item0)
	}

	taskList.PushFront(item)
	item = taskList.PopFront().(FileChunk)
	if item.Start != item0.Start {
		t.Log("taskList PushFront is incorrect")
		t.Error(item, item0)
	}
}

func TestRadio(t *testing.T) {
	var taskList = RadioTaskList{
		make([]RadioChunk, 0, 100),
		sync.Mutex{},
	}

	var item = FileChunk{
		0,
		20,
	}
	item0 := item
	item1 := item
	item2 := item
	item3 := item
	item0.Start = 0
	item1.Start = 20
	item2.Start = 40
	item3.Start = 60

	log.Println(taskList.Tasks())
	appendToPendings(item0, &taskList)
	appendToPendings(item1, &taskList)
	appendToPendings(item2, &taskList)
	appendToPendings(item3, &taskList)
	log.Println(taskList.Tasks())

	item = taskList.PopFront().(FileChunk)

	if item.Start != 0 || item.Length != 80 {
		t.Log("appendToPendings is incorrect")
		t.Error(&taskList)
	}

	item4 := item
	item4.Start = 80
	item4.Length = 8000000
	appendToPendings(item, &taskList)
	appendToPendings(item4, &taskList)
	log.Println(taskList.Tasks())

}
