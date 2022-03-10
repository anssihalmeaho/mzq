package msg

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

// MessageServer represents messaging server
type MessageServer struct {
	Opt      Options
	Listener net.Listener
	Conns    map[string]net.Conn
	lock     sync.RWMutex
	recChan  chan Msg
}

// Options contains options for messaging server
type Options struct {
	Addr string
}

func (server *MessageServer) addConn(conn net.Conn) {
	server.lock.Lock()
	defer server.lock.Unlock()

	server.Conns[conn.RemoteAddr().String()] = conn
}

func (server *MessageServer) removeConn(addr string) {
	server.lock.Lock()
	defer server.lock.Unlock()

	delete(server.Conns, addr)
}

func (server *MessageServer) getConn(addr string) (net.Conn, bool) {
	server.lock.RLock()
	defer server.lock.RUnlock()

	conn, found := server.Conns[addr]
	return conn, found
}

func (server *MessageServer) receiver(conn net.Conn) {
	server.addConn(conn)
	remoteAddr := conn.RemoteAddr().String()
	defer server.removeConn(remoteAddr)

	for {
		recData, err := bufio.NewReader(conn).ReadBytes(0)
		if errors.Is(err, net.ErrClosed) {
			break
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(fmt.Sprintf("Error in reading: %v", err))
			return
		}
		msg := Msg{
			FromAddr: remoteAddr,
			Data:     string(recData[:len(recData)-1]),
		}

		// non-blocking send
		select {
		case server.recChan <- msg:
		default:
		}
	}
}

func (server *MessageServer) acceptor() {
	for {
		conn, err := server.Listener.Accept()
		if err != nil {
			panic(err)
		}
		go server.receiver(conn)
	}
}

// CreateServer creates new messaging server
func CreateServer(options Options) (*MessageServer, error) {
	server := &MessageServer{
		Opt:     options,
		Conns:   make(map[string]net.Conn),
		recChan: make(chan Msg, 10),
	}
	ln, err := net.Listen("tcp", options.Addr)
	if err != nil {
		return nil, err
	}
	server.Listener = ln

	go server.acceptor()
	return server, nil
}

// Msg represents message
type Msg struct {
	FromAddr string
	Data     string
}

// Connection represents one (TCP) connection
type Connection struct {
	Conn      net.Conn
	ServerRef *MessageServer
}

// OpenConnection opens new connection towards given address
func (server *MessageServer) OpenConnection(addr string) (*Connection, error) {
	conn, found := server.getConn(addr)
	if found {
		connection := &Connection{
			Conn:      conn,
			ServerRef: server,
		}
		return connection, nil
	}
	var err error
	conn, err = net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	go server.receiver(conn)
	connection := &Connection{
		Conn:      conn,
		ServerRef: server,
	}
	return connection, nil
}

// Receive message
func (server *MessageServer) Receive() (Msg, error) {
	msg := <-server.recChan
	return msg, nil
}

// Send sends message to connection
func (con *Connection) Send(data string) error {
	b := append([]byte(data), 0)
	_, err := con.Conn.Write(b)
	return err
}

// Close connection
func (con *Connection) Close() {
	con.Conn.Close()
}
