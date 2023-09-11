package oviewer

import (
	"bytes"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestRoot_toggleColumnMode(t *testing.T) {
	tests := []struct {
		name       string
		columnMode bool
	}{
		{
			name:       "test1",
			columnMode: false,
		},
		{
			name:       "test2",
			columnMode: true,
		},
	}
	tcellNewScreen = fakeScreen
	defer func() {
		tcellNewScreen = tcell.NewScreen
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := NewRoot(bytes.NewBufferString("test"))
			if err != nil {
				t.Fatal(err)
			}
			root.Doc.ColumnMode = tt.columnMode
			root.toggleColumnMode()
			if root.Doc.ColumnMode == tt.columnMode {
				t.Errorf("root.toggleColumnMode() = %v, want %v", root.Doc.ColumnMode, !tt.columnMode)
			}
		})
	}
}

func Test_position(t *testing.T) {
	type args struct {
		height int
		str    string
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "test1",
			args: args{
				height: 30,
				str:    "1",
			},
			want: 1,
		},
		{
			name: "test.5",
			args: args{
				height: 30,
				str:    ".5",
			},
			want: 15,
		},
		{
			name: "test20%",
			args: args{
				height: 30,
				str:    "20%",
			},
			want: 6,
		},
		{
			name: "test.3",
			args: args{
				height: 45,
				str:    "30%",
			},
			want: 13.5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := docPosition(tt.args.height, tt.args.str); got != tt.want {
				t.Errorf("position() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_jumpPosition(t *testing.T) {
	type args struct {
		height int
		str    string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "test1",
			args: args{
				height: 30,
				str:    "1",
			},
			want: 1,
		},
		{
			name: "test.3",
			args: args{
				height: 10,
				str:    ".3",
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jumpPosition(tt.args.height, tt.args.str); got != tt.want {
				t.Errorf("jumpPosition() = %v, want %v", got, tt.want)
			}
		})
	}
}
