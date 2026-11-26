package main

import (
	"reflect"
	"testing"
)

func TestCastValueList(t *testing.T) {
	cases := []struct {
		in   string
		want []any
	}{
		{"tcpdump|strace|IP routing", []any{"tcpdump", "strace", "IP routing"}},
		{" a | b ", []any{"a", "b"}},
		{"strace: syscalls|x, y", []any{"strace: syscalls", "x, y"}}, // ':' and ',' survive
		{"", []any{}},              // explicit empty list
		{"a||b|", []any{"a", "b"}}, // empty items dropped
	}
	for _, c := range cases {
		got, err := castValue(c.in, "list")
		if err != nil {
			t.Fatalf("castValue(%q, list) error: %v", c.in, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("castValue(%q, list) = %#v, want %#v", c.in, got, c.want)
		}
	}
}
