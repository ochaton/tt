package install_ee

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_stringQueue_Insert(t *testing.T) {
	type fields struct {
		mtx *sync.Mutex
		buf []string
	}
	type args struct {
		data string
	}
	testBuf := make([]string, 0)
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "Simple insert",
			fields: fields{mtx: &sync.Mutex{}, buf: testBuf},
			args:   args{data: "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &stringQueue{
				mtx: tt.fields.mtx,
				buf: tt.fields.buf,
			}
			q.Insert(tt.args.data)
			require.Len(t, q.buf, 1)
		})
	}
}

func Test_stringQueue_InsertBatch(t *testing.T) {
	type fields struct {
		mtx *sync.Mutex
		buf []string
	}
	type args struct {
		data []string
	}
	testBuf := make([]string, 0)
	testInsertBuf := []string{"test1", "test2"}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "Simple batch insert",
			fields: fields{mtx: &sync.Mutex{}, buf: testBuf},
			args:   args{data: testInsertBuf},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &stringQueue{
				mtx: tt.fields.mtx,
				buf: tt.fields.buf,
			}
			q.InsertBatch(tt.args.data)
			require.Len(t, q.buf, 2)
			if q.buf[0] != "test1" || q.buf[1] != "test2" {
				t.Errorf("Missing values in the queue buffer after batch insert")
			}
		})
	}
}

func Test_stringQueue_Pop(t *testing.T) {
	type fields struct {
		mtx *sync.Mutex
		buf []string
	}
	testBuf1 := []string{"test"}
	testBuf2 := []string{"test1", "test2"}
	testBuf3 := make([]string, 0)
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:   "Simple pop from queue with 1 element",
			fields: fields{mtx: &sync.Mutex{}, buf: testBuf1},
			want:   "test",
			wantErr: assert.ErrorAssertionFunc(func(t assert.TestingT, err error,
				i ...interface{}) bool {
				return true
			}),
		},
		{
			name:   "Simple pop from queue with several elements",
			fields: fields{mtx: &sync.Mutex{}, buf: testBuf2},
			want:   "test1",
			wantErr: assert.ErrorAssertionFunc(func(t assert.TestingT, err error,
				i ...interface{}) bool {
				return true
			}),
		},
		{
			name:   "Simple pop from queue with no elements",
			fields: fields{mtx: &sync.Mutex{}, buf: testBuf3},
			want:   "",
			wantErr: assert.ErrorAssertionFunc(func(t assert.TestingT, err error,
				i ...interface{}) bool {
				return err.Error() == emptyQueueError
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &stringQueue{
				mtx: tt.fields.mtx,
				buf: tt.fields.buf,
			}
			got, err := q.Pop()
			if !tt.wantErr(t, err, fmt.Sprintf("Pop()")) {
				return
			}
			assert.Equalf(t, tt.want, got, "Pop()")
		})
	}
}

func Test_stringQueue_GetAll(t *testing.T) {
	type fields struct {
		mtx *sync.Mutex
		buf []string
	}
	testBuf1 := []string{"test1", "test2"}
	testBuf2 := make([]string, 0)
	tests := []struct {
		name    string
		fields  fields
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:   "Simple GetAll from queue",
			fields: fields{mtx: &sync.Mutex{}, buf: testBuf1},
			want:   testBuf1,
			wantErr: assert.ErrorAssertionFunc(func(t assert.TestingT, err error,
				i ...interface{}) bool {
				return true
			}),
		},
		{
			name:   "Simple GetAll from queue with no elements",
			fields: fields{mtx: &sync.Mutex{}, buf: testBuf2},
			want:   nil,
			wantErr: assert.ErrorAssertionFunc(func(t assert.TestingT, err error,
				i ...interface{}) bool {
				return err.Error() == emptyQueueError
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &stringQueue{
				mtx: tt.fields.mtx,
				buf: tt.fields.buf,
			}
			got, err := q.GetAll()
			if !tt.wantErr(t, err, fmt.Sprintf("GetAll()")) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetAll()")
		})
	}
}

func Test_stringQueue_Len(t *testing.T) {
	type fields struct {
		mtx *sync.Mutex
		buf []string
	}
	testBuf1 := []string{"test1", "test2"}
	testBuf2 := make([]string, 0)
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "Non empty queue",
			fields: fields{mtx: &sync.Mutex{}, buf: testBuf1},
			want:   len(testBuf1),
		},
		{
			name:   "Empty queue",
			fields: fields{mtx: &sync.Mutex{}, buf: testBuf2},
			want:   len(testBuf2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &stringQueue{
				mtx: tt.fields.mtx,
				buf: tt.fields.buf,
			}
			assert.Equalf(t, tt.want, q.Len(), "Len()")
		})
	}
}
