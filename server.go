// Package coap provides a CoAP client and server.
package coap

import (
	"log"
	"net"
	"time"
)

const maxPktLen = 1500

// Handler is a type that handles CoAP messages.
type Handler interface {
	// Handle the message and optionally return a response message.
	ServeCOAP(c *Conn, m Message) Message
}

type funcHandler func(c *Conn, m Message) Message

func (f funcHandler) ServeCOAP(c *Conn, m Message) Message {
	return f(c, m)
}

// FuncHandler builds a handler from a function.
func FuncHandler(f func(c *Conn, m Message) Message) Handler {
	return funcHandler(f)
}

//should handlePacket be exported?
func handlePacket(c *Conn, data []byte, addr Addr,
	rh Handler) {

	msg, err := ParseDgramMessage(data)
	if err != nil {
		log.Printf("Error parsing %v", err)
		return
	}

	rv := rh.ServeCOAP(c, msg)
	if rv != nil {
		Transmit(c, addr, rv)
	}
}

// Transmit a message.
func Transmit(c *Conn, address Addr, m Message) error {
	d, err := m.MarshalBinary()
	if err != nil {
		return err
	}
	net, err := c.Network()
	if err != nil {
		return err
	}
	if net == "udp" {
		addr := address.Udp.String()

		if string([]byte(addr)) == "<nil>" {
			_, err = c.conn.Write(d)
		} else {
			//	_, err = c.conn.Write(d) //this line is just to prevent the "use of writeto with pre-connected connection" error
			_, err = c.conn.WriteToUDP(d, address.Udp)
		}
		return err
	}
	if net == "tcp" {
		_, err := c.connTCP.Write(d)
		return err
	}
	return err
}

// Receive a message.
func Receive(c *Conn, buf []byte) (Message, error) {
	n, err := c.Network()
	if err != nil {
		return nil, err
	}
	switch n {
	case "udp":
		c.conn.SetReadDeadline(time.Now().Add(ResponseTimeout))

		nr, err := c.conn.Read(buf)
		if err != nil {
			return &DgramMessage{}, err
		}
		return ParseDgramMessage(buf[:nr])
	case "tcp":
		c.connTCP.SetReadDeadline(time.Now().Add(ResponseTimeout))
		for {
			_, err := c.connTCP.Read(buf)
			if err != nil {
				return &TcpMessage{}, err
			}
			m, _, err := PullTcp(buf)
			return m, err
		}
	default:
		return nil, err

	}
}

// ListenAndServe binds to the given address and serve requests forever. This has not been modified to handle TCP
func ListenAndServe(n, addr string, rh Handler) error {
	uaddr, err := net.ResolveUDPAddr(n, addr)
	if err != nil {
		return err
	}

	l, err := net.ListenUDP(n, uaddr)
	if err != nil {
		return err
	}

	return Serve(
		&Conn{conn: l},
		rh,
	)
}

// Serve processes incoming UDP packets on the given listener, and processes
// these requests forever (or until the listener is closed).
func Serve(listener *Conn, rh Handler) error {
	buf := make([]byte, maxPktLen)
	n, err := listener.Network()
	if err != nil {
		return err
	}
	if n == "udp" {
		for {
			nr, addr, err := listener.conn.ReadFromUDP(buf)
			if err != nil {
				if neterr, ok := err.(net.Error); ok && (neterr.Temporary() || neterr.Timeout()) {
					time.Sleep(5 * time.Millisecond)
					continue
				}
				return err
			}
			tmp := make([]byte, nr)
			copy(tmp, buf)
			go handlePacket(listener, tmp, Addr{Udp: addr}, rh)
		}
	}
	if n == "tcp" { //i need to get this function to keep looping and reading until it gets a full TCP packet
		for {
			_, err := listener.connTCP.Read(listener.buf) //maybe needs pullTCP()?
			if err != nil {
				if neterr, ok := err.(net.Error); ok && (neterr.Temporary() || neterr.Timeout()) {
					time.Sleep(5 * time.Millisecond)
					continue
				}
				return err
			}
			if len(listener.buf) > 0 {

				tmp, buf, err := PullTcp(listener.buf)
				if err != nil {
					return err
				}
				if len(listener.buf) > len(buf) {
					listener.buf = buf

					m, err := tmp.MarshalBinary()
					if err != nil {
						return err
					}

					addr, err := net.ResolveTCPAddr("tcp", listener.connTCP.RemoteAddr().String())
					if err != nil {
						return err
					}
					go handlePacket(listener, m, Addr{Tcp: addr}, rh)
				}
			}
		}
	}
	return err
}
