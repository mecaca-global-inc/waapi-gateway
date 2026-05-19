package api

import "testing"

func TestHashKeyDeterministic(t *testing.T) {
	a := hashKey("hello")
	b := hashKey("hello")
	if a != b {
		t.Fatal("hash should be deterministic")
	}
	if hashKey("hello") == hashKey("world") {
		t.Fatal("different inputs should hash differently")
	}
}

func TestSubtleEqual(t *testing.T) {
	if !subtleEqual("abc", "abc") {
		t.Fatal("equal strings should match")
	}
	if subtleEqual("abc", "abd") {
		t.Fatal("different strings should not match")
	}
}
