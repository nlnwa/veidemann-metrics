package rethinkdb

import "testing"

func TestContains(t *testing.T) {
	type args struct {
		list []string
		item []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"1", args{list: []string{"foo", "bar", "abc", "def"}, item: []string{"foo"}}, true},
		{"2", args{list: []string{"foo", "bar", "abc", "def"}, item: []string{"bar"}}, true},
		{"3", args{list: []string{"foo", "bar", "abc", "def"}, item: []string{"bar", "def"}}, true},
		{"4", args{list: []string{"foo", "bar", "abc", "def"}, item: []string{"bar", "foo"}}, true},
		{"5", args{list: []string{"foo", "bar", "abc", "def"}, item: []string{"xyz"}}, false},
		{"6", args{list: []string{"foo", "bar", "abc", "def"}, item: []string{"bar", "xyz"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.args.list, tt.args.item...); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
