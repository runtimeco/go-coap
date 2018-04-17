package coap

import (
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	// ResponseTimeout is the amount of time to wait for a
	// response.
	ResponseTimeout = time.Second * 2
	// ResponseRandomFactor is a multiplier for response backoff.
	ResponseRandomFactor = 1.5
	// MaxRetransmit is the maximum number of times a message will
	// be retransmitted.
	MaxRetransmit = 4
)

// Conn is a CoAP client connection.
type Conn struct {
	connTCP *net.TCPConn
	buf     []byte
	conn    *net.UDPConn
	Net     string
}

type Addr struct {
	Tcp *net.TCPAddr
	Udp *net.UDPAddr
}

// Dial connects a CoAP client.
func Dial(n, addr string) (*Conn, error) {
	switch n {
	case "udp":
		uaddr, err := net.ResolveUDPAddr(n, addr)
		if err != nil {
			return nil, err
		}

		s, err := net.DialUDP("udp", nil, uaddr)
		if err != nil {
			return nil, err
		}

		return &Conn{conn: s, buf: make([]byte, maxPktLen), connTCP: nil}, nil
	case "tcp":
		taddr, err := net.ResolveTCPAddr(n, addr)
		if err != nil {
			return nil, err
		}

		s, err := net.DialTCP("tcp", nil, taddr)
		if err != nil {
			return nil, err
		}

		return &Conn{conn: nil, buf: make([]byte, maxPktLen), connTCP: s}, nil
	default:
		return nil, errors.New("unrecognized network type")
	}
}

// Send a message.  Get a response if there is one.
func (c *Conn) Send(req Message) (Message, error) {

	//defer c.Close()
	//not sure if that's a good idea to have it be default behavior. Maybe have it be based on a setting in Conn?
	err := Transmit(c, Addr{}, req)
	if err != nil {
		return nil, err
	}

	if !req.IsConfirmable() {
		return nil, nil
	}
	fmt.Println("about to receive in send()")
	rv, err := Receive(c, c.buf)
	if err != nil {
		return nil, err
	}

	return rv, nil
}

// Receive a message.
func (c *Conn) Receive() (Message, error) {
	rv, err := Receive(c, c.buf)
	if err != nil {
		return nil, err
	}
	return rv, nil
}

func (c *Conn) Network() (string, error) {
	fmt.Println("conn.Network() called")
	if c.Net != "" {
		return c.Net, nil
	}
	if c.conn != nil && c.connTCP != nil {
		fmt.Println("satisfied conditions for udp/tcp both being non-nil")
		return "", errors.New("multiple non-nil connections in Conn. it should be only one")
	}
	if c.conn != nil {
		return "udp", nil
	}
	if c.connTCP != nil {
		return "tcp", nil
	} else {
		fmt.Println("both connections are nil")
		return "", errors.New("all connections in Conn struct are nil")
	}
}

func (c *Conn) Close() error {
	n, err := c.Network()
	if err != nil {
		return err
	}
	switch n {
	case "udp":
		return c.conn.Close()
	case "tcp":
		return c.connTCP.Close()
	}
	return err
}
