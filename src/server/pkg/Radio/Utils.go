package Radio

import (
	"encoding/hex"
	xxhash "github.com/cespare/xxhash"
	"github.com/dustin/randbo"
	"log"
	"server/pkg/BufferedFile"
	"server/pkg/Socket"
	"strconv"
)

const (
	CHUNK_SIZE          int64 = 1024 * 400 // Bytes
	MAX_CHUNKS_IN_QUEUE       = 2048       // which means there shuold be 2048 RadioChunk instances in pending queue at most
	SEND_INTERVAL             = 500        // check pending list every 800ms to send new items
)

func (r *RadioTaskList) Tasks() *[]RadioChunk {
	return &(r.tasks)
}

func (r *RadioTaskList) Length() int {
	return len(r.tasks)
}

func (r *RadioTaskList) Append(chunks []RadioChunk) {
	r.tasks = append(r.tasks, chunks...)
}

func (r *RadioTaskList) PushBack(chunks []RadioChunk) {
	r.Append(chunks)
}

func (r *RadioTaskList) PopBack() RadioChunk {
	var bottomItem = r.tasks[len(r.tasks)-1]
	r.tasks = r.tasks[:len(r.tasks)-1]
	return bottomItem
}

func (r *RadioTaskList) PopFront() RadioChunk {
	var item = r.tasks[0]
	r.tasks = r.tasks[1:len(r.tasks)]
	return item
}

func (r *RadioTaskList) PushFront(chunk RadioChunk) {
	r.tasks = append(r.tasks, chunk)
	copy(r.tasks[1:], r.tasks[0:])
	r.tasks[0] = chunk
}

func splitChunk(chunk FileChunk) []RadioChunk {
	var result_queue = make([]RadioChunk, 0)
	var real_chunk_size = CHUNK_SIZE
	var chunks = chunk.Length / real_chunk_size

	// keep chunks in a reasonable amount
	for chunks > MAX_CHUNKS_IN_QUEUE {
		real_chunk_size *= 2
		chunks = chunk.Length / real_chunk_size
	}
	var c_pos = chunk.Start
	for i := 0; int64(i) < chunks; i++ {
		result_queue = append(result_queue, FileChunk{c_pos, real_chunk_size})
		c_pos += real_chunk_size
	}

	if chunk.Length%real_chunk_size > 0 {
		result_queue = append(result_queue, FileChunk{c_pos, chunk.Length % real_chunk_size})
	}

	return result_queue
}

func pushLargeChunk(chunk FileChunk, queue *RadioTaskList) {
	var new_items = splitChunk(chunk)
	queue.Append(new_items)
}

func pushRamChunk(chunk RAMChunk, queue *RadioTaskList) {
	// re-split chunk in ram won't save any memory, so just make it in queue
	queue.Append([]RadioChunk{chunk})
}

func appendToPendings(chunk RadioChunk, list *RadioTaskList) {
	list.locker.Lock()
	defer list.locker.Unlock()
	switch chunk.(type) {
	case RAMChunk:
		pushRamChunk(chunk.(RAMChunk), list)
		return
	}
	var chunkF = chunk.(FileChunk)

	if list.Length() > 0 {
		var bottomItem = list.PopBack()
		switch bottomItem.(type) {
		case FileChunk:
			var bottomItemF = bottomItem.(FileChunk)
			// try to merge new chunk into old chunk
			var new_length = bottomItemF.Length + chunkF.Length
			if bottomItemF.Start+bottomItemF.Length == chunkF.Start { // if two chunks are neighbor
				// concat two chunks and re-split them
				pushLargeChunk(FileChunk{bottomItemF.Start, new_length}, list)
			} else { // or just push those in
				list.Append([]RadioChunk{bottomItemF})                       // push the old chunk back
				pushLargeChunk(FileChunk{chunkF.Start, chunkF.Length}, list) // and new one
			}
		case RAMChunk:
			// special RadioRAMChunk should be considered
			// TODO: merge RadioRAMChunk if possible
			list.Append([]RadioChunk{bottomItem}) // put it back, since we don't merge anything now
			pushLargeChunk(chunkF, list)
		}
	} else {
		pushLargeChunk(FileChunk{chunkF.Start, chunkF.Length}, list)
	}

	if list.Length() >= MAX_CHUNKS_IN_QUEUE*2 {
		// TODO: add another function to re-split chunks in queue
		log.Println("There're ", list.Length(), "chunks in a single queue!")
	}
}

func fetchAndSend(client *Socket.SocketClient, list *RadioTaskList, file *BufferedFile.BufferedFile) error {
	list.locker.Lock()
	defer list.locker.Unlock()
	if list.Length() <= 0 {
		return nil
	}

	var item = list.PopFront()

	switch item.(type) {
	case FileChunk:
		var item = item.(FileChunk)
		var buf = make([]byte, item.Length)
		length, err := file.ReadAt(buf, item.Start)
		if int64(length) != item.Length || err != nil {
			// move back
			list.PushFront(item)
			return nil
		}
		_, err = client.WriteRaw(buf)
		return err
	case RAMChunk:
		_, err := client.WriteRaw(item.(RAMChunk).Data)
		return err
	}
	return nil
}

func genArchiveSign(name string) string {
	var buf = make([]byte, 16)
	randbo.New().Read(buf)
	hash := xxhash.Sum64String(name + hex.EncodeToString(buf))
	return strconv.FormatUint(hash, 32)
}
