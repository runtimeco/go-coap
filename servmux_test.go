package coap

import (
	"testing"
)

func TestPathMatching(t *testing.T) {
	m := NewServeMux()

	msgs := map[string]int{}
	//using nil for network type because no transport is being used in this test
	m.HandleFunc("", "/a", func(c *Conn, m Message) Message {
		msgs["a"]++
		t.Log("get a request on /a ", string(m.Payload()))
		return nil
	})
	m.HandleFunc("", "/b", func(c *Conn, m Message) Message {
		msgs["b"]++
		t.Log("get a request on /b ", string(m.Payload()))
		return nil
	})

	msg := &DgramMessage{}
	cTcp := &Conn{Net: "tcp"} //it's easier to set Conn.Net and not use it than it is to explicitly accept connections without a stated transport type
	cUdp := &Conn{Net: "udp"}
	msg.SetPathString("/a")
	msg.SetPayload([]byte("hi a1"))
	m.ServeCOAP(cTcp, msg)
	msg.SetPathString("/a")
	msg.SetPayload([]byte("hi a2"))
	m.ServeCOAP(cTcp, msg)
	msg.SetPathString("/b")
	msg.SetPayload([]byte("hi b1"))
	m.ServeCOAP(cUdp, msg)
	msg.SetPathString("/c")
	msg.SetPayload([]byte("hi c"))
	m.ServeCOAP(cUdp, msg)
	msg.MessageBase.typ = NonConfirmable
	msg.SetPathString("/c")
	msg.SetPayload([]byte("hi c"))
	m.ServeCOAP(cTcp, msg)

	if msgs["a"] != 2 {
		t.Errorf("Expected 2 messages for /a, got %v", msgs["a"])
	}
	if msgs["b"] != 1 {
		t.Errorf("Expected 1 message for /b, got %v", msgs["b"])
	}
}

func TestPathMatch(t *testing.T) {
	tests := []struct {
		pattern, path string
		exp           bool
	}{
		{"", "", false},
		{"/a/b/c", "/a/b/c", true},
		{"/a/b/c", "/a/b/c/d", false},
		{"/a/b/c/", "/a/b/c/d", true},
		{"/a/b/c", "/", false},
		{"/a/", "/", false},
	}

	for _, test := range tests {
		if pathMatch(test.pattern, test.path) != test.exp {
			t.Errorf("Failed on pathMatch(%q, %q), wanted %v",
				test.pattern, test.path, test.exp)
		}
	}
}
