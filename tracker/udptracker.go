package tracker

import (
	"encoding/binary"
	"log"
	"math/rand"
	"net"
	"time"
)

type connectionPacket struct {
	connectionID  uint64
	action        uint32
	transactionID uint32
}

type announcePacket struct {
	connection connectionPacket
	infoHash   [20]byte //The info-hash of the torrent you want announce yourself in.
	peerID     [20]byte //Your peer id.
	downloaded uint64   //The number of byte you've downloaded in this session.
	left       uint64   //The number of bytes you have left to download until you're finished.
	uploaded   uint64   //	The number of bytes you have uploaded in this session.
	event      uint32
	ip         uint32 //Your ip address. Set to 0 if you want the tracker to use the sender of this UDP packet.
	key        uint32 //A unique key that is randomized by the client.
	numWant    int32  //The maximum number of peers you want in the reply. Use -1 for default.
	port       uint16 //The port you're listening on.
	extensions uint16
	expayload  [9]byte
}

type announceResponsePacket struct {
	Action        uint32
	TransactionID uint32
	Interval      uint32
	Leechers      uint32
	Seeders       uint32
	Peers         []Peer
}

func unmarshallAnnounce(buffer []byte) *announceResponsePacket {

	response := announceResponsePacket{
		Action:        binary.BigEndian.Uint32(buffer[0:4]),
		TransactionID: binary.BigEndian.Uint32(buffer[4:8]),
		Interval:      binary.BigEndian.Uint32(buffer[8:12]),
		Leechers:      binary.BigEndian.Uint32(buffer[12:16]),
		Seeders:       binary.BigEndian.Uint32(buffer[16:20]),
	}

	peers := []Peer{}

	for i := 20; i < len(buffer); i += 6 {
		peer := new(Peer)
		peer.IP = net.IP(buffer[i : i+4])
		peer.Port = binary.BigEndian.Uint16(buffer[i+4 : i+6])
		peers = append(peers, *peer)
	}

	response.Peers = peers

	return &response
}

type UDPTracker struct {
	Host         string
	conn         *net.UDPConn
	connectionID uint64
	InfoHash     [20]byte
	Length       uint64
}

func (t *UDPTracker) sendReceiveMessage(message interface{}) ([]byte, int, error) {
	buf := StructToBuffer(message)

	_, err := t.conn.Write(buf)

	if err != nil {
		return nil, 0, err
	}

	buffer := make([]byte, 2048)
	n, _, err := t.conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, 0, err
	}

	return buffer[0:n], n, nil
}

func (t *UDPTracker) Connect() (e error) {
	s, err := net.ResolveUDPAddr("udp4", t.Host)
	c, err := net.DialUDP("udp4", nil, s)
	t.conn = c
	t.conn.SetReadDeadline(time.Now().Add(time.Second * 5))

	if err != nil {
		return err
	}

	log.Printf("Handshacking UDP server %s (%s)\n", t.Host, c.RemoteAddr().String())

	handshake := &connectionPacket{
		connectionID:  0x41727101980,
		action:        0,
		transactionID: rand.Uint32(),
	}

	buffer, n, err := t.sendReceiveMessage(handshake)
	if err != nil {
		return err
	}

	t.connectionID = binary.BigEndian.Uint64(buffer[8:n])
	return
}

func (t *UDPTracker) Announce() (peers []Peer, e error) {

	conn := &connectionPacket{
		connectionID:  t.connectionID,
		action:        1,
		transactionID: rand.Uint32(),
	}

	var peerID [20]byte
	copy(peerID[:], "-SHOToTorrent-0.1---")

	var exPayload [9]byte
	copy(exPayload[:], "/announce")

	announce := announcePacket{
		connection: *conn,
		infoHash:   t.InfoHash,
		peerID:     peerID,
		downloaded: 0,
		left:       t.Length,
		uploaded:   0,
		event:      2,
		ip:         0,
		key:        rand.Uint32(),
		numWant:    200,
		port:       0x64ab,
		extensions: 521,
		expayload:  exPayload,
	}

	buffer, _, err := t.sendReceiveMessage(announce)
	if err != nil {
		return nil, err
	}

	response := unmarshallAnnounce(buffer)
	log.Printf("Tracker Answered with %d peers\n", len(response.Peers))

	return response.Peers, nil

}
