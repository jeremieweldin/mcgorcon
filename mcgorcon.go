// Package mcgorcon is a Minecraft RCON Client written in Go.
// It is designed to be easy to use and integrate into your own applications.
package mcgorcon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

type packetType int32

// Client is a representation of an RCON client.
type Client struct {
	password   string
	connection net.Conn
}

// header is the header of a Minecraft RCON packet.
type header struct {
	Size       int32
	RequestID  int32
	PacketType packetType
}

const packetTypeCommand packetType = 2
const packetTypeAuth packetType = 3
const requestIDBadLogin int32 = -1

// Dial up the server and establish a RCON conneciton.
func Dial(host string, port int, pass string) Client {
	// Combine the host and port to form the address.
	address := host + ":" + fmt.Sprint(port)
	// Actually establish the conneciton.
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		panic(err)
	}
	// Create the client object, since the connection has been established.
	c := Client{password: pass, connection: conn}
	// TODO - server validation to make sure we're talking to a real RCON server.
	// For now, just return the client and assume it's a real server.
	return c
}

// SendCommand sends a command to the server and returns the result (often nothing).
func (c *Client) SendCommand(command string) string {
	// Because I'm lazy, just authenticate with every command.
	c.authenticate()
	// Send the packet.
	head, payload := c.sendPacket(packetTypeCommand, []byte(command))
	// Auth was bad, panic.
	if head.RequestID == requestIDBadLogin {
		panic("NO AITH")
	}
	return string(payload)
}

// authenticate authenticates the user with the server.
func (c *Client) authenticate() {
	// Send the packet.
	head, _ := c.sendPacket(packetTypeAuth, []byte(c.password))
	// If the credentials were bad, panic.
	if head.RequestID == requestIDBadLogin {
		panic("BAD AUTH")
	}
}

// sendPacket sends the binary packet representation to the server and returns the response.
func (c *Client) sendPacket(t packetType, p []byte) (header, []byte) {
	// Generate the binary packet.
	packet := packetise(t, p)
	// Send the packet over the wire.
	_, err := c.connection.Write(packet)
	if err != nil {
		panic("WRITE FAIL")
	}
	// Receive and decode the response.
	head, payload := depacketise(c.connection)
	return head, payload
}

// packetise encodes the packet type and payload into a binary representation to send over the wire.
func packetise(t packetType, p []byte) []byte {
	// Generate a random request ID.
	pad := [2]byte{}
	length := int32(len(p) + 10)
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, length)
	binary.Write(&buf, binary.LittleEndian, int32(0))
	binary.Write(&buf, binary.LittleEndian, t)
	binary.Write(&buf, binary.LittleEndian, p)
	binary.Write(&buf, binary.LittleEndian, pad)
	// Notchian server doesn't like big packets :(
	if buf.Len() >= 1460 {
		panic("Packet too big when packetising.")
	}
	// Return the bytes.
	return buf.Bytes()
}

// depacketise decodes the binary packet into a native Go struct.
func depacketise(r io.Reader) (header, []byte) {
	head := header{}
	err := binary.Read(r, binary.LittleEndian, &head)
	if err != nil {
		panic(err)
	}
	payload := make([]byte, head.Size-8)
	_, err = io.ReadFull(r, payload)
	if err != nil {
		panic(err)
	}
	return head, payload[:len(payload)-2]
}
