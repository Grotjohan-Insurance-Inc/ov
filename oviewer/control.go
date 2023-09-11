package oviewer

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru/v2"
)

// controlSpecifier represents a control request.
type controlSpecifier struct {
	searcher Searcher
	request  request
	chunkNum int
	done     chan bool
}

// request represents a control request.
type request string

// control requests.
const (
	requestStart    request = "start"
	requestContinue request = "continue"
	requestFollow   request = "follow"
	requestClose    request = "close"
	requestReload   request = "reload"
	requestLoad     request = "load"
	requestSearch   request = "search"
)

// ControlFile controls file read and loads in chunks.
// ControlFile can be reloaded by file name.
func (m *Document) ControlFile(file *os.File) error {
	m.setNewLoadChunks()

	go func() {
		atomic.StoreInt32(&m.closed, 0)
		r, err := m.fileReader(file)
		if err != nil {
			atomic.StoreInt32(&m.closed, 1)
			log.Println(err)
		}
		atomic.StoreInt32(&m.eof, 0)
		reader := bufio.NewReader(r)
		for sc := range m.ctlCh {
			reader, err = m.controlFile(sc, reader)
			if err != nil {
				log.Println(sc.request, err)
			}
			if sc.done != nil {
				if err != nil {
					sc.done <- false
				} else {
					sc.done <- true
				}
				close(sc.done)
			}
		}
		log.Println("close m.ctlCh")
	}()

	m.requestStart()
	return nil
}

// ControlReader is the controller for io.Reader.
// Assuming call from Exec. reload executes the argument function.
func (m *Document) ControlReader(r io.Reader, reload func() *bufio.Reader) error {
	m.setNewLoadChunks()
	m.seekable = false
	reader := bufio.NewReader(r)

	go func() {
		var err error
		for sc := range m.ctlCh {
			reader, err = m.controlReader(sc, reader, reload)
			if err != nil {
				log.Println(sc.request, err)
			}
			if sc.done != nil {
				if err != nil {
					sc.done <- false
				} else {
					sc.done <- true
				}
				close(sc.done)
			}
		}
		log.Println("close ctlCh")
	}()

	m.requestStart()
	return nil
}

// ControlLog controls log.
// ControlLog is only supported reload.
func (m *Document) ControlLog() error {
	go func() {
		for sc := range m.ctlCh {
			m.controlLog(sc)
			if sc.done != nil {
				sc.done <- true
				close(sc.done)
			}
		}
		log.Println("close m.ctlCh")
	}()
	return nil
}

// controlFile controls file read and loads in chunks.
// controlFile receives and executes request.
func (m *Document) controlFile(sc controlSpecifier, reader *bufio.Reader) (*bufio.Reader, error) {
	if atomic.LoadInt32(&m.closed) == 1 && sc.request != requestReload {
		return nil, fmt.Errorf("%w %s", ErrAlreadyClose, sc.request)
	}
	var err error
	switch sc.request {
	case requestStart:
		return m.firstRead(reader)
	case requestContinue:
		if !m.seekable && LoadChunksLimit > 0 {
			if m.loadedChunks.Len() >= LoadChunksLimit {
				return reader, ErrOverChunkLimit
			}
		}
		return m.continueRead(reader)
	case requestFollow:
		return m.followRead(reader)
	case requestLoad:
		return m.loadRead(reader, sc.chunkNum)
	case requestSearch:
		return m.searchRead(reader, sc.chunkNum, sc.searcher)
	case requestReload:
		if !m.WatchMode {
			m.loadedChunks.Purge()
		}
		reader, err = m.reloadRead(reader)
		m.requestStart()
		return reader, err
	case requestClose:
		err = m.close()
		log.Println(err)
		return reader, err
	default:
		panic(fmt.Sprintf("unexpected %s", sc.request))
	}
}

// controlReader controls io.Reader.
// controlReader receives and executes request.
func (m *Document) controlReader(sc controlSpecifier, reader *bufio.Reader, reload func() *bufio.Reader) (*bufio.Reader, error) {
	switch sc.request {
	case requestStart:
		// controlReader is the same for first and continue.
		return m.continueRead(reader)
	case requestContinue:
		return m.continueRead(reader)
	case requestLoad:
		m.currentChunk = sc.chunkNum
		m.managesChunksMem(sc.chunkNum)
	case requestReload:
		if reload != nil {
			log.Println("reload")
			reader = reload()
			m.requestStart()
		}
	default:
		panic(fmt.Sprintf("unexpected %s", sc.request))
	}
	return reader, nil
}

// controlLog controls log.
// controlLog receives and executes request.
func (m *Document) controlLog(sc controlSpecifier) {
	switch sc.request {
	case requestLoad:
	case requestReload:
		m.reset()
	default:
		panic(fmt.Sprintf("unexpected %s", sc.request))
	}
}

// unloadChunk unloads the chunk from memory.
func (m *Document) unloadChunk(chunkNum int) {
	m.loadedChunks.Remove(chunkNum)
	m.chunks[chunkNum].lines = nil
}

// managesChunksFile manages Chunks of regular files.
// manage chunk eviction.
func (m *Document) managesChunksFile(chunkNum int) error {
	if chunkNum == 0 {
		return nil
	}
	for m.loadedChunks.Len() > FileLoadChunksLimit {
		k, _, _ := m.loadedChunks.GetOldest()
		if chunkNum != k {
			m.unloadChunk(k)
		}
	}

	chunk := m.chunks[chunkNum]
	if len(chunk.lines) != 0 || atomic.LoadInt32(&m.closed) != 0 {
		return fmt.Errorf("%w %d", ErrAlreadyLoaded, chunkNum)
	}
	return nil
}

// managesChunksMem manages Chunks other than regular files.
// The specified chunk is already in memory, so only the first chunk is unloaded.
// Change the start position after unloading.
func (m *Document) managesChunksMem(chunkNum int) {
	if chunkNum == 0 {
		return
	}
	if (LoadChunksLimit < 0) || (m.loadedChunks.Len() < LoadChunksLimit) {
		return
	}
	k, _, _ := m.loadedChunks.GetOldest()
	m.unloadChunk(k)
	m.mu.Lock()
	m.startNum = (k + 1) * ChunkSize
	m.mu.Unlock()
}

// setNewLoadChunks creates a new LRU cache.
// Manage chunks loaded in LRU cache.
func (m *Document) setNewLoadChunks() {
	capacity := FileLoadChunksLimit + 1
	if !m.seekable {
		if LoadChunksLimit > 0 {
			capacity = LoadChunksLimit + 1
		}
	}
	chunks, err := lru.New[int, struct{}](capacity)
	if err != nil {
		log.Printf("lru new %s", err)
	}
	m.loadedChunks = chunks
}

// requestStart send instructions to start reading.
func (m *Document) requestStart() {
	go func() {
		m.ctlCh <- controlSpecifier{
			request: requestStart,
		}
	}()
}

// requestContinue sends instructions to continue reading.
func (m *Document) requestContinue() {
	go func() {
		m.ctlCh <- controlSpecifier{
			request: requestContinue,
		}
	}()
}

// requestLoad sends instructions to load chunks into memory.
func (m *Document) requestLoad(chunkNum int) {
	sc := controlSpecifier{
		request:  requestLoad,
		chunkNum: chunkNum,
		done:     make(chan bool),
	}
	m.ctlCh <- sc
	<-sc.done
}

// requestSearch sends instructions to load chunks into memory.
func (m *Document) requestSearch(chunkNum int, searcher Searcher) bool {
	sc := controlSpecifier{
		request:  requestSearch,
		searcher: searcher,
		chunkNum: chunkNum,
		done:     make(chan bool),
	}
	m.ctlCh <- sc
	return <-sc.done
}

// requestClose sends instructions to close the file.
func (m *Document) requestClose() {
	atomic.StoreInt32(&m.readCancel, 1)
	sc := controlSpecifier{
		request: requestClose,
		done:    make(chan bool),
	}

	log.Println("close send")
	m.ctlCh <- sc
	<-sc.done
	atomic.StoreInt32(&m.readCancel, 0)
}

// requestReload sends instructions to reload the file.
func (m *Document) requestReload() {
	sc := controlSpecifier{
		request: requestReload,
		done:    make(chan bool),
	}
	m.ctlCh <- sc
	<-sc.done
}
