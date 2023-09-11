package oviewer

import (
	"reflect"
	"sync/atomic"
	"testing"
)

func Test_store_chunkRange(t *testing.T) {
	t.Parallel()
	type fields struct {
		chunks   []*chunk
		startNum int32
		endNum   int32
	}
	type args struct {
		chunkNum int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
		want1  int
	}{
		{
			name: "test0",
			fields: fields{
				chunks: []*chunk{
					NewChunk(0),
				},
				startNum: 0,
				endNum:   10,
			},
			args:  args{chunkNum: 0},
			want:  0,
			want1: 10,
		},
		{
			name: "test1Chunk",
			fields: fields{
				chunks: []*chunk{
					NewChunk(0),
					NewChunk(0),
					NewChunk(0),
				},
				startNum: 0,
				endNum:   20010,
			},
			args:  args{chunkNum: 1},
			want:  0,
			want1: 10000,
		},
		{
			name: "test2Chunk",
			fields: fields{
				chunks: []*chunk{
					NewChunk(0),
					NewChunk(0),
					NewChunk(0),
				},
				startNum: 0,
				endNum:   20010,
			},
			args:  args{chunkNum: 2},
			want:  0,
			want1: 10,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &store{
				chunks:   tt.fields.chunks,
				startNum: tt.fields.startNum,
				endNum:   tt.fields.endNum,
			}
			got, got1 := s.chunkRange(tt.args.chunkNum)
			if got != tt.want {
				t.Errorf("store.chunkRange() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("store.chunkRange() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func testNewStore(t *testing.T, chunkNum int, capacity int) *store {
	t.Helper()
	s := NewStore()
	s.chunks = make([]*chunk, chunkNum)
	t.Logf("capacity %d", capacity)
	s.setNewLoadChunks(capacity)
	for i := 0; i < chunkNum; i++ {
		chunk := NewChunk(0)
		chunk.lines = make([][]byte, ChunkSize)
		for j := 0; j < ChunkSize; j++ {
			chunk.lines[j] = []byte("a")
		}
		s.chunks[i] = chunk
		s.loadChunksMem(i)
	}
	return s
}

func Test_store_swapChunksFile(t *testing.T) {
	t.Parallel()
	type fields struct {
		maxChunks int
	}
	type args struct {
		chunkNums []int
		loaded    int
	}
	tests := []struct {
		name     string
		capacity int
		fields   fields
		args     args
		want     bool
	}{
		{
			name:     "test0",
			fields:   fields{maxChunks: 100},
			capacity: 100,
			args: args{
				chunkNums: []int{0},
				loaded:    0,
			},
			want: true,
		},
		{
			name:     "test1",
			fields:   fields{maxChunks: 100},
			capacity: 100,
			args: args{
				chunkNums: []int{1},
				loaded:    1,
			},
			want: true,
		},
		{
			name:     "testFalse",
			fields:   fields{maxChunks: 100},
			capacity: 100,
			args: args{
				chunkNums: []int{99},
				loaded:    1,
			},
			want: false,
		},
		{
			name:     "testEvict",
			fields:   fields{maxChunks: 100},
			capacity: 3,
			args: args{
				chunkNums: []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
				loaded:    2,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := testNewStore(t, tt.fields.maxChunks, tt.capacity)
			for _, num := range tt.args.chunkNums {
				s.swapLoadedFile(num)
			}
			if got := s.isLoadedChunk(tt.args.loaded, true); got != tt.want {
				t.Logf("loadedChunks: %v", s.loadedChunks.Len())
				t.Errorf("store.swapChunksFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_store_isContinueRead(t *testing.T) {
	t.Parallel()
	type args struct {
		limit int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "testOK",
			args: args{
				limit: 100,
			},
			want: true,
		},
		{
			name: "testNoLimit",
			args: args{
				limit: -1,
			},
			want: true,
		},
		{
			name: "testNG",
			args: args{
				limit: 9,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := testNewStore(t, 10, tt.args.limit)
			if got := s.isContinueRead(tt.args.limit); got != tt.want {
				t.Logf("loadedChunks: %v", s.loadedChunks.Len())
				t.Errorf("store.isContinueRead() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_store_loadChunksMem(t *testing.T) {
	t.Parallel()
	type fields struct {
		maxChunks int
	}
	type args struct {
		chunkNums []int
		contains  int
	}
	tests := []struct {
		name   string
		limit  int
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "test0",
			fields: fields{maxChunks: 3},
			limit:  3,
			args: args{
				chunkNums: []int{0},
				contains:  0,
			},
			want: false,
		},
		{
			name:   "test1",
			fields: fields{maxChunks: 3},
			limit:  10,
			args: args{
				chunkNums: []int{1},
				contains:  1,
			},
			want: true,
		},
		{
			name:   "test2",
			fields: fields{maxChunks: 3},
			limit:  3,
			args: args{
				chunkNums: []int{1, 2, 3},
				contains:  2,
			},
			want: true,
		},
		{
			name:   "testNoLimit",
			fields: fields{maxChunks: 10},
			limit:  -1,
			args: args{
				chunkNums: []int{1},
				contains:  1,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := testNewStore(t, tt.fields.maxChunks, tt.limit)
			for _, chunkNum := range tt.args.chunkNums {
				s.loadChunksMem(chunkNum)
			}
			if got := s.loadedChunks.Contains(tt.args.contains); got != tt.want {
				t.Logf("loadedChunks: %v[%v]", s.loadedChunks.Keys(), tt.args.contains)
				t.Errorf("store.loadChunksMem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_store_evictChunksMem(t *testing.T) {
	t.Parallel()
	type fields struct {
		maxChunks int
	}
	type args struct {
		chunkNums []int
		current   int
	}
	tests := []struct {
		name   string
		limit  int
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "test0",
			fields: fields{maxChunks: 3},
			limit:  3,
			args: args{
				chunkNums: []int{0},
				current:   0,
			},
			want: false,
		},
		/*
			{
				name:   "test1",
				fields: fields{maxChunks: 3},
				limit:  10,
				args: args{
					chunkNums: []int{1},
					current:   1,
				},
				want: true,
			},
		*/
		{
			name:   "testCurrent",
			fields: fields{maxChunks: 10},
			limit:  3,
			args: args{
				chunkNums: []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
				current:   9,
			},
			want: true,
		},
		{
			name:   "testOld",
			fields: fields{maxChunks: 10},
			limit:  3,
			args: args{
				chunkNums: []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
				current:   6,
			},
			want: false,
		},
		{
			name:   "testNoLimit",
			fields: fields{maxChunks: 10},
			limit:  -1,
			args: args{
				chunkNums: []int{1},
				current:   1,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := testNewStore(t, tt.fields.maxChunks, tt.limit)
			for _, chunkNum := range tt.args.chunkNums {
				s.loadChunksMem(chunkNum)
				s.evictChunksMem(chunkNum)
			}
			s.evictChunksMem(tt.args.current)
			if got := s.loadedChunks.Contains(tt.args.current); got != tt.want {
				t.Logf("loadedChunks: %v", s.loadedChunks.Keys())
				t.Errorf("store.swapChunksFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testChunkStore(t *testing.T) (*store, *chunk) {
	t.Helper()
	s := NewStore()
	chunk := s.chunkForAdd(false, 0)
	s.appendLine(chunk, []byte("a"))
	s.appendLine(chunk, []byte("b"))
	s.appendLine(chunk, []byte("c"))
	return s, chunk
}

func Test_store_appendLine(t *testing.T) {
	t.Parallel()
	type appendArgs struct {
		noNewlineEOF int32
		line         []byte
	}
	tests := []struct {
		name string
		args []appendArgs
		want []byte
	}{
		{
			name: "test0",
			args: []appendArgs{
				{
					noNewlineEOF: 0,
					line:         []byte("pre"),
				},
				{
					noNewlineEOF: 0,
					line:         []byte("hello"),
				},
			},
			want: []byte("hello"),
		},
		{
			name: "testJoin",
			args: []appendArgs{
				{
					noNewlineEOF: 0,
					line:         []byte("hel"),
				},
				{
					noNewlineEOF: 1,
					line:         []byte("lo"),
				},
			},
			want: []byte("hello"),
		},
		{
			name: "testJoin2",
			args: []appendArgs{
				{
					noNewlineEOF: 0,
					line:         []byte("hel"),
				},
				{
					noNewlineEOF: 1,
					line:         []byte("lo\n"),
				},
			},
			want: []byte("hello\n"),
		},
		{
			name: "testJoinBlank",
			args: []appendArgs{
				{
					noNewlineEOF: 0,
					line:         []byte("hello"),
				},
				{
					noNewlineEOF: 1,
					line:         []byte(""),
				},
			},
			want: []byte("hello"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, chunk := testChunkStore(t)

			for _, app := range tt.args {
				atomic.StoreInt32(&s.noNewlineEOF, app.noNewlineEOF)
				s.appendLine(chunk, app.line)
			}
			if got := chunk.lines[len(chunk.lines)-1]; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("store.appendLine() = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}

func Test_store_chunkForAdd(t *testing.T) {
	t.Parallel()
	type fields struct {
		chunks   []*chunk
		startNum int
		endNum   int
	}
	type args struct {
		isFile bool
		start  int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *chunk
	}{
		{
			name: "test0",
			fields: fields{
				chunks:   []*chunk{},
				startNum: 0,
				endNum:   0,
			},
			args: args{
				isFile: false,
				start:  0,
			},
			want: NewChunk(0),
		},
		{
			name: "testMem",
			fields: fields{
				chunks: []*chunk{
					NewChunk(0),
					NewChunk(0),
					NewChunk(0),
				},
				startNum: 0,
				endNum:   1,
			},
			args: args{
				isFile: false,
				start:  0,
			},
			want: NewChunk(0),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewStore()
			s.chunks = tt.fields.chunks
			if got := s.chunkForAdd(tt.args.isFile, tt.args.start); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("store.chunkForAdd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_store_isLoadedChunk(t *testing.T) {
	t.Parallel()
	type fields struct {
		chunks int
	}
	type args struct {
		chunkNum int
		isFile   bool
	}
	tests := []struct {
		name   string
		limit  int
		fields fields
		args   args
		want   bool
	}{
		{
			name:  "test0",
			limit: 100,
			fields: fields{
				chunks: 0,
			},
			args: args{
				chunkNum: 0,
				isFile:   true,
			},
			want: true,
		},
		{
			name:  "test1",
			limit: 100,
			fields: fields{
				chunks: 3,
			},
			args: args{
				chunkNum: 1,
				isFile:   true,
			},
			want: true,
		},
		{
			name:  "testTrue",
			limit: 10,
			fields: fields{
				chunks: 3,
			},
			args: args{
				chunkNum: 99,
				isFile:   false,
			},
			want: true,
		},
		{
			name:  "testFail",
			limit: 10,
			fields: fields{
				chunks: 3,
			},
			args: args{
				chunkNum: 99,
				isFile:   true,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := testNewStore(t, tt.fields.chunks, tt.limit)
			if got := s.isLoadedChunk(tt.args.chunkNum, tt.args.isFile); got != tt.want {
				t.Logf("isLoadedChunk %v", s.loadedChunks.Keys())
				t.Errorf("store.isLoadedChunk() = %v, want %v", got, tt.want)
			}
		})
	}
}
