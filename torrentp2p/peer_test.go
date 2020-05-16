package torrentp2p

import (
	"bytes"
	"net"
	"testing"

	torrent "github.com/vaguilera/MiniTorrent/torrentfile"
)

type connStub struct {
	net.Conn
	buff []byte
}

func (c *connStub) Write(b []byte) (n int, err error) {
	c.buff = b
	return len(b), nil
}

func Test_setBitField(t *testing.T) {

	torrent := torrent.Torrent{
		PieceHashes: make([][20]byte, 10),
	}
	data := []byte{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xA0}
	peer := NewPeer(&torrent, nil, nil)

	peer.setBitField(data)

	var expected byte = 1
	for _, cbit := range peer.bitfield {
		if cbit != expected {
			t.Errorf("Expected %d, got %d", expected, cbit)
		}
		if expected == 1 {
			expected = 0
		} else {
			expected = 1
		}
	}
}

func Test_sendUint32(t *testing.T) {
	peer := NewPeer(&torrent.Torrent{}, nil, nil)
	Cs := &connStub{}
	peer.conn = Cs
	peer.sendUint32(444)
	res := bytes.Compare([]byte{0, 0, 1, 188}, Cs.buff)

	if res != 0 {
		t.Errorf("Expected [0 0 1 188], got %v", Cs.buff)
	}

}

func Test_unMarshallHandShake(t *testing.T) {

	data := []byte{10}
	torrent := torrent.Torrent{}

	reserved := [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
	infoHash := [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0, 0, 0, 0, 0, 0, 0, 0}
	data = append(data, "BitTorrent protocol"...)
	data = append(data, reserved[:]...)
	data = append(data, infoHash[:]...)
	data = append(data, "PeerIDPeerIDPeerIDPe"...)

	peer := NewPeer(&torrent, nil, nil)
	hands := peer.unMarshallHandShake(data)

	if hands.ptrLength != 10 {
		t.Errorf("Expected 10, got %d", hands.ptrLength)
	}
	if string(hands.protocol[:]) != "BitTorrent protocol" {
		t.Errorf("Expected 'BitTorrent protocol', got %s", hands.protocol)
	}
	if hands.reserved != reserved {
		t.Errorf("Expected %v, got %v", hands.reserved, reserved)
	}

	if hands.infoHash != infoHash {
		t.Errorf("Expected %v, got %v", hands.infoHash, infoHash)
	}

	if string(hands.peerID[:]) != "PeerIDPeerIDPeerIDPe" {
		t.Errorf("Expected 'PeerIDPeerIDPeerIDPe', got %s", hands.peerID)
	}

}
