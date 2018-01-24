package amigo

import (
	"fmt"
	"testing"
)

func TestPrefixToString(t *testing.T) {
	prefix := kadPrefix{
		bytes:       []byte{0xF0, 0xB1},
		lastByteLen: 5,
	}

	t.Log(prefix.String())
}

func TestPrefixGrow(t *testing.T) {
	pfx := NewKadPrefix()

	pfx = pfx.grow(1)
	pfx = pfx.grow(0)
	pfx = pfx.grow(1)
	pfx = pfx.grow(1)

	fmt.Printf("prefix: %+v\n", pfx.bytes)

	if pfx.String() != "1011" {
		t.Fatalf("pfx should be 1011 after growth, but got %s", pfx.String())
	}
}
