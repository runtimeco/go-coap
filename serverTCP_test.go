package coap

import (
	"net"
	"testing"
)

func startTCPLisenter(t *testing.T) (*net.TCPListener, string) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:5683")
	if err != nil {
		t.Fatal("Can't resolve TCP addr")
	}
	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		t.Fatal("Can't listen on TCP: ", err)
	}

	coapServerAddr := tcpListener.Addr().String()
	return tcpListener, coapServerAddr
}

func dialAndSendTCP(t *testing.T, addr string, req Message) Message {
	c, err := Dial("tcp", addr)
	if err != nil {
		t.Log("the addr to dial was: ", addr)
		t.Fatalf("Error dialing: %v", err)
	}
	m, err := c.Send(req)
	if err != nil {
		t.Fatalf("Error sending request: %v", err)
	}
	return m
}

func TestServeTCPWithAckResponse(t *testing.T) {
	req := &TcpMessage{
		MessageBase{
			typ:       Confirmable,
			code:      POST,
			messageID: 9876,
			payload:   []byte("Content sent by client"),
		},
	}
	req.SetOption(ContentFormat, TextPlain)
	req.SetPathString("/req/path")

	res := &TcpMessage{
		MessageBase{
			typ:       Acknowledgement,
			code:      Content,
			messageID: req.MessageID(),
			payload:   []byte("Reply from CoAP server"),
		},
	}
	res.SetOption(ContentFormat, TextPlain)
	res.SetPath(req.Path())

	handler := FuncHandler(func(c *Conn, m Message) Message {
		t.Log(m.Type(), "payload:", m.Payload())

		assertEqualMessages(t, req, m)
		return res
	})

	tcpListener, coapServerAddr := startTCPLisenter(t)
	defer tcpListener.Close()
	go dialAndTest(t, coapServerAddr, req, true, res)

	tcpConn, err := tcpListener.AcceptTCP()
	if err != nil {
		t.Fatal("err accepting TCPconn: ", err)
	}

	go Serve(
		&Conn{connTCP: tcpConn},
		handler,
	)

	/*	m := dialAndSendTCP(t, coapServerAddr, req)

		if m == nil {
			t.Fatalf("Didn't receive CoAP response")
		}
		assertEqualMessages(t, res, m)
	*/
}

func TestServeTCPWithoutAckResponse(t *testing.T) {
	req := &TcpMessage{
		MessageBase{
			typ:       NonConfirmable,
			code:      POST,
			messageID: 54321,
			payload:   []byte("Content sent by client"),
		},
	}
	req.SetOption(ContentFormat, AppOctets)

	handler := FuncHandler(func(c *Conn, m Message) Message {
		assertEqualMessages(t, req, m)
		return nil
	})

	tcpListener, coapServerAddr := startTCPLisenter(t)
	defer tcpListener.Close()
	dialAndTest(t, coapServerAddr, req, false, &TcpMessage{})
	tcpConn, err := tcpListener.AcceptTCP()
	if err != nil {
		t.Fatal("err accepting TCPconn: ", err)
	}

	go Serve(
		&Conn{connTCP: tcpConn},
		handler,
	)

}

func dialAndTest(t *testing.T, addr string, req *TcpMessage, ack bool, res *TcpMessage) {
	m := dialAndSendTCP(t, addr, req)
	if ack {
		assertEqualMessages(t, res, m)

	} else if m != nil {
		t.Errorf("recieved an ack when expecting none")
	}

}
