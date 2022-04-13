package bro

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/anssihalmeaho/mzq/msg"
	"github.com/anssihalmeaho/mzq/queue"
)

const debugPrintingOn = false

func debugPrint(args ...interface{}) {
	if debugPrintingOn {
		fmt.Println(args...)
	}
}

type peerState int

const (
	stateDown peerState = iota
	stateUp
	//stateClosing
	stateClosed
)

type peerInfo struct {
	name    string
	addr    string
	recAddr string
	conn    *msg.Connection
	state   peerState
}

// Broker ...
type Broker struct {
	OwnName   string
	OwnAddr   string
	Server    *msg.MessageServer
	Peers     *PeerStore
	RegCh     chan queueReg
	PayloadCh chan payloadMsg
	Decoder   func([]byte) interface{}
}

type payloadMsg struct {
	queueName string
	data      []byte
}

type queueReg struct {
	name    string
	q       *queue.Queue
	replyCh chan error
	remove  bool
}

// PeerStore ...
type PeerStore struct {
	peers []*peerInfo
	sync.RWMutex
}

func newPeerStore() *PeerStore {
	return &PeerStore{peers: []*peerInfo{}}
}

func (ps *PeerStore) getPrint() string {
	ps.RLock()
	defer ps.RUnlock()

	s := "\n"
	for i, v := range ps.peers {
		s += fmt.Sprintf("\n  %d: addr: %s, name: %s, recaddr: %s, conn: %#v, state: %v", i, v.addr, v.name, v.recAddr, v.conn, v.state)
	}
	return s + "\n"
}

func (ps *PeerStore) getPeerByName(name string) (*msg.Connection, error) {
	for _, v := range ps.peers {
		if v.name == name {
			if v.state == stateUp {
				return v.conn, nil
			}
			return nil, fmt.Errorf("Invalid connection state (%v)", v.state)
		}
	}
	return nil, fmt.Errorf("Peer (%s) not found", name)
}

func (ps *PeerStore) updClosing() []*msg.Connection {
	result := []*msg.Connection{}
	ps.Lock()
	defer ps.Unlock()

	for _, v := range ps.peers {
		switch v.state {
		case stateDown /*stateClosing,*/, stateClosed:
		case stateUp:
			v.state = stateClosed
			result = append(result, v.conn)
		}
	}
	return result
}

func (ps *PeerStore) addPeer(addr string, conn *msg.Connection) int {
	ps.Lock()
	defer ps.Unlock()

	peer := &peerInfo{
		addr: addr,
		conn: conn,
	}
	ps.peers = append(ps.peers, peer)
	return len(ps.peers) - 1
}

func (ps *PeerStore) updLeaving(name string) (*msg.Connection, bool) {
	ps.Lock()
	defer ps.Unlock()

	for _, v := range ps.peers {
		if v.name == name {
			v.state = stateClosed
			return v.conn, true
		}
	}
	return nil, false
}

func (ps *PeerStore) updConn2(conID int, name, addr string) (*msg.Connection, bool) {
	ps.Lock()
	defer ps.Unlock()

	if conID+1 > len(ps.peers) {
		return nil, false
	}
	if (ps.peers[conID].name != "") && (ps.peers[conID].name != name) {
		return nil, false
	}
	ps.peers[conID].name = name
	ps.peers[conID].recAddr = addr
	ps.peers[conID].state = stateUp
	return ps.peers[conID].conn, true
}

func (ps *PeerStore) updConn(conn *msg.Connection, name, addr string) (*msg.Connection, bool) {
	ps.Lock()
	defer ps.Unlock()

	for i := range ps.peers {
		if ps.peers[i].name == name {
			ps.peers[i].name = name
			ps.peers[i].conn = conn
			ps.peers[i].recAddr = addr
			ps.peers[i].state = stateUp
			return ps.peers[i].conn, true
		}
	}
	peer := &peerInfo{
		// addr: addr,
		conn:    conn,
		name:    name,
		recAddr: addr,
		state:   stateUp,
	}
	ps.peers = append(ps.peers, peer)
	return conn, true
}

type msgFormat struct {
	MsgName     string          `json:"msg-name"`
	TargetQName string          `json:"qname"`
	Data        json.RawMessage `json:"data"`
	PayloadData []byte          `json:"pdata"`
}

type connectMsg struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type connectAckMsg struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type leaveMsg struct {
	Name string `json:"name"`
}

// RegisterQueue ...
func (broker *Broker) RegisterQueue(qname string, q *queue.Queue) error {
	replyCh := make(chan error)
	req := queueReg{
		name:    qname,
		q:       q,
		replyCh: replyCh,
	}
	broker.RegCh <- req
	return <-replyCh
}

// UnRegisterQueue ...
func (broker *Broker) UnRegisterQueue(qname string) error {
	replyCh := make(chan error)
	req := queueReg{
		name:    qname,
		replyCh: replyCh,
		remove:  true,
	}
	broker.RegCh <- req
	return <-replyCh
}

// Close ...
func (broker *Broker) Close() {
	conns := broker.Peers.updClosing()

	for _, conn := range conns {
		leave := &leaveMsg{
			Name: broker.OwnName,
		}
		leaveData, err := json.Marshal(leave)
		if err != nil {
			debugPrint("Marshal failed: ", err)
			continue
		}
		leaveMsg := &msgFormat{
			MsgName: "leave",
			Data:    leaveData,
		}
		leaveMsgData, err := json.Marshal(leaveMsg)
		if err != nil {
			debugPrint("Marshal failed: ", err)
			continue
		}
		err = conn.Send(string(leaveMsgData))
		if err != nil {
			debugPrint("Leave send failed: ", err)
			continue
		}
	}
}

func (broker *Broker) manager() {
	queues := map[string]*queue.Queue{}

	for {
		select {
		// register/unregister queue with name
		case reg := <-broker.RegCh:
			if reg.remove {
				delete(queues, reg.name)
			} else {
				queues[reg.name] = reg.q
			}
			reg.replyCh <- nil

		// payload message from receiver
		case message := <-broker.PayloadCh:
			if message.queueName == "" {
				debugPrint("Empty queue name")
				continue
			}
			q, found := queues[message.queueName]
			if !found {
				continue
			}
			var qitem interface{}
			if broker.Decoder != nil {
				qitem = broker.Decoder(message.data)
			} else {
				qitem = message.data
			}
			isFull := q.PutNoWait(qitem)
			if isFull {
				debugPrint("Queue full, dropping")
			}
		}
	}
}

// SendMsg ...
func (broker *Broker) SendMsg(nodeName, queueName string, data []byte) error {
	if nodeName == broker.OwnName {
		// its local queue
		broker.PayloadCh <- payloadMsg{queueName: queueName, data: data}
		return nil
	}

	// ok, lets send it to some peer node
	con, err := broker.Peers.getPeerByName(nodeName)
	payloadmsg := &msgFormat{
		MsgName:     "payload",
		TargetQName: queueName,
		PayloadData: data,
	}
	payloadMsgData, err := json.Marshal(payloadmsg)
	if err != nil {
		errval := fmt.Errorf("Marshal failed: %v", err)
		debugPrint(errval)
		return errval
	}
	err = con.Send(string(payloadMsgData))
	if err != nil {
		errval := fmt.Errorf("Message send failed: %v", err)
		debugPrint(errval)
		return errval
	}
	return nil
}

func (broker *Broker) receiver() {
	for {
		msg, err := broker.Server.Receive()
		if err != nil {
			debugPrint("receiver error: ", err)
			continue
		}
		//fmt.Println(fmt.Sprintf("MSG: %#v", msg))

		var msgform msgFormat
		if err := json.Unmarshal([]byte(msg.Data), &msgform); err != nil {
			debugPrint("decode error: ", err)
			continue
		}

		switch msgform.MsgName {

		// connect received
		case "connect":
			//fmt.Println("connect received")

			var conMsg connectMsg
			if err := json.Unmarshal([]byte(msgform.Data), &conMsg); err != nil {
				debugPrint("Connect msg decode failed: ", err)
				continue
			}

			conn, err := broker.Server.OpenConnection(msg.FromAddr)
			con, found := broker.Peers.updConn(conn, conMsg.Name, msg.FromAddr)
			if !found {
				debugPrint(fmt.Sprintf("Connection not found (%d)", conMsg.ID))
				//fmt.Println("PEERS: ", broker.Peers.getPrint())
				continue
			}
			//fmt.Println("PEERS: ", broker.Peers.getPrint())

			connectAckData := &connectAckMsg{
				Name: broker.OwnName,
				ID:   conMsg.ID,
			}
			conAckData, err := json.Marshal(connectAckData)
			if err != nil {
				debugPrint("Marshal failed: ", err)
				continue
			}
			connectAckmsg := &msgFormat{
				MsgName: "connect-ack",
				Data:    conAckData,
			}
			conAckMsgData, err := json.Marshal(connectAckmsg)
			if err != nil {
				debugPrint("Marshal failed: ", err)
				continue
			}
			err = con.Send(string(conAckMsgData))
			if err != nil {
				debugPrint("Ack send failed: ", err)
				continue
			}

		// connect-ack received
		case "connect-ack":
			//fmt.Println("connect-ack received")

			var conAckMsg connectAckMsg
			if err := json.Unmarshal(msgform.Data, &conAckMsg); err != nil {
				debugPrint("Connect-ack msg decode failed: ", err)
				continue
			}
			_, found := broker.Peers.updConn2(conAckMsg.ID, conAckMsg.Name, msg.FromAddr)
			if !found {
				debugPrint(fmt.Sprintf("Connection not found (%d)(%s)", conAckMsg.ID, conAckMsg.Name))
				//fmt.Println("PEERS: ", broker.Peers.getPrint())
				continue
			}
			//fmt.Println("PEERS: ", broker.Peers.getPrint())

		// leave received
		case "leave":
			var leaveMsg leaveMsg
			if err := json.Unmarshal(msgform.Data, &leaveMsg); err != nil {
				debugPrint("Leave msg decode failed: ", err)
				continue
			}
			//fmt.Println("LEAVE RECEIVED: ", leaveMsg.Name)
			con, found := broker.Peers.updLeaving(leaveMsg.Name)
			if !found {
				debugPrint("Connection not found: ", leaveMsg.Name)
			}
			con.Close()

		// payload message
		case "payload":
			broker.PayloadCh <- payloadMsg{queueName: msgform.TargetQName, data: msgform.PayloadData}
		}
	}
}

// CreateBroker creates broker instance
func CreateBroker(options map[string]interface{}) (*Broker, error) {
	return CreateBrokerV2(options, nil)
}

// CreateBrokerV2 creates broker instance
func CreateBrokerV2(options map[string]interface{}, decoder func([]byte) interface{}) (*Broker, error) {
	namev, namefound := options["own-name"]
	if !namefound {
		return nil, fmt.Errorf("Own name not found")
	}
	ownname := namev.(string)
	addrv, found := options["own-addr"]
	if !found {
		return nil, fmt.Errorf("Own address not found")
	}
	ownAddr, addrok := addrv.(string)
	if !addrok {
		return nil, fmt.Errorf("Invalid format for own address")
	}

	v, found := options["addrs"]
	if !found {
		return nil, fmt.Errorf("No peer addresses found")
	}
	peers, ok := v.([]string)
	if !ok {
		return nil, fmt.Errorf("Invalid format for peers")
	}

	// create own msg server
	server, err := msg.CreateServer(msg.Options{Addr: ownAddr})
	if err != nil {
		return nil, fmt.Errorf("CreateServer failed: %v", err)
	}

	broker := &Broker{
		OwnName:   ownname,
		OwnAddr:   ownAddr,
		Server:    server,
		Peers:     newPeerStore(),
		RegCh:     make(chan queueReg),
		PayloadCh: make(chan payloadMsg),
		Decoder:   decoder,
	}
	go broker.receiver()
	go broker.manager()

	// connect to peers
	for _, addr := range peers {
		con, err := server.OpenConnection(addr)
		if err != nil {
			debugPrint("Connecting failed: ", err)
			continue
		}

		id := broker.Peers.addPeer(addr, con)

		connectData := &connectMsg{
			Name: ownname,
			ID:   id,
		}
		conData, err := json.Marshal(connectData)
		if err != nil {
			debugPrint("Marshal failed: ", err)
			continue
		}
		connectmsg := &msgFormat{
			MsgName: "connect",
			Data:    conData,
		}
		conMsgData, err := json.Marshal(connectmsg)
		if err != nil {
			debugPrint("Marshal failed: ", err)
			continue
		}
		err = con.Send(string(conMsgData))
		if err != nil {
			debugPrint("Initial send failed: ", err)
			continue
		}
	}

	return broker, nil
}
