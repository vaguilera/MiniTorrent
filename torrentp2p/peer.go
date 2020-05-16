package torrentp2p

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/vaguilera/MiniTorrent/torrentfile"
	"github.com/vaguilera/MiniTorrent/tracker"
)

const (
	CHOKE = iota
	UNCHOKE
	INTERESTED
	NOT_INTERESTED
	HAVE
	BITFIELD
	REQUEST
	PIECE
	CANCEL
	PORT      // Not implemented. For DHT support.
	EXTENSION = 20
)

type handshakeP struct {
	ptrLength byte
	protocol  [19]byte
	reserved  [8]byte
	infoHash  [20]byte
	peerID    [20]byte
}

type Peer struct {
	host             tracker.Peer
	conn             net.Conn
	torrent          *torrentfile.Torrent
	chocked          bool
	currentPieceData []byte
	currentPieceNum  uint32
	bytesRcvd        uint32
	bytesReq         uint32
	currentPieceSize uint32
	bitfield         []byte
	peersQueue       chan tracker.Peer
	resultsChan      chan StPieceResult
	bitFieldRecv     bool
	currentInfoPiece StPiece
	pieceLength      uint32
	status           byte
}

func NewPeer(torrent *torrentfile.Torrent, peersQueue chan tracker.Peer, results chan StPieceResult) *Peer {
	p := &Peer{
		chocked:     true,
		torrent:     torrent,
		bitfield:    make([]byte, len(torrent.PieceHashes)),
		peersQueue:  peersQueue,
		resultsChan: results,
	}
	p.currentPieceData = make([]byte, torrent.PieceLength)
	return p
}

func (p *Peer) unMarshallHandShake(buffer []byte) *handshakeP {
	var response handshakeP

	response.ptrLength = buffer[0]
	copy(response.protocol[:], buffer[1:20])
	copy(response.reserved[:], buffer[20:28])
	copy(response.infoHash[:], buffer[28:48])
	copy(response.peerID[:], buffer[48:68])

	return &response
}

func (p *Peer) sendUint32(n uint32) (err error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, n)
	_, err = p.conn.Write(buf)
	return
}

func (p *Peer) sendMessage(messageID byte, payload []byte) error {

	var buf []byte

	if payload == nil {
		buf = []byte{0, 0, 0, 1, messageID}
	} else {
		l := uint32(len(payload)) + 1
		buf = make([]byte, l)
		buf[0] = messageID
		copy(buf[1:], payload)
		p.sendUint32(l)
	}
	_, err := p.conn.Write(buf)
	return err
}

func (p *Peer) processBlock(payload []byte) {
	//piece := binary.BigEndian.Uint32(payload[0:4])
	offset := binary.BigEndian.Uint32(payload[4:8])
	data := payload[8:]
	copy(p.currentPieceData[offset:], data)
	p.bytesRcvd += uint32(len(data))

}

func (p *Peer) sendPieceRequest() error {
	var payload [12]byte
	var err error

	blocksize := uint32(0x4000)
	if (p.bytesReq + 0x4000) > p.currentPieceSize {
		blocksize = p.currentPieceSize - p.bytesReq
	}

	binary.BigEndian.PutUint32(payload[0:], p.currentPieceNum)
	binary.BigEndian.PutUint32(payload[4:], p.bytesReq)
	binary.BigEndian.PutUint32(payload[8:], blocksize)
	err = p.sendMessage(REQUEST, payload[:])
	if err != nil {
		return err
	}
	p.bytesReq += blocksize
	return nil
}

func (p *Peer) setBitField(bitfield []byte) {
	pos := 0
	var bitmask byte

out:
	for _, cbyte := range bitfield {
		bitmask = 128
		for bitmask >= 1 {
			cbit := (cbyte & bitmask)
			if cbit > 0 {
				p.bitfield[pos] = 1
			}
			pos++
			if pos == len(p.bitfield) {
				break out
			}
			bitmask /= 2
		}
	}

	p.bitFieldRecv = true
}

func (p *Peer) processMessage(msg Message) error {

	strHost := p.host.IP.String()
	switch msg.ID {

	case CHOKE:
		log.Printf("(%s) CHOKE\n", strHost)
		p.chocked = true
	case UNCHOKE:
		log.Printf("(%s) UNCHOKE\n", strHost)
		p.chocked = false
		if p.status == 1 {
			p.status = 2
		}
	case HAVE:
		piece := binary.BigEndian.Uint32(msg.Payload)
		log.Printf("(%s) HAVE %d\n", strHost, piece)
		p.bitfield[piece] = 1
	case BITFIELD:
		log.Printf("(%s) BITFIELD\n", strHost)
		if p.status > 0 {
			return errors.New("(%s) BITFIELD received but not as first message")
		}
		p.setBitField(msg.Payload)
		p.status = 1
		if p.chocked {
			log.Printf("(%s) Sending UNCHOKE\n", strHost)
			p.sendMessage(UNCHOKE, nil)
			p.sendMessage(INTERESTED, nil)
		}

	case INTERESTED:
		log.Printf("(%s) INTERESTED\n", strHost)
	case NOT_INTERESTED:
		log.Printf("(%s) NOT INTERESTED\n", strHost)
	case PIECE:
		p.processBlock(msg.Payload)
		if (p.bytesReq < p.currentPieceSize) && (p.chocked == false) {
			return p.sendPieceRequest()
		}
	default:
		log.Printf("Undefined or unexpected message %d - %v\n", msg.ID, msg.Payload)
	}
	return nil
}

func (p *Peer) readMessage(msgQueue chan<- Message, out chan<- struct{}) {

	for {
		lengthBuf := make([]byte, 4)
		_, err := io.ReadFull(p.conn, lengthBuf)
		if err != nil {
			log.Printf("Error reading from peer: %s\n", err)
			close(out)
			return
		}

		lengthM := binary.BigEndian.Uint32(lengthBuf)
		if lengthM == 0 {
			log.Printf("(%s) Keep alive message\n", p.host.IP.String())
			return
		}
		messageBuf := make([]byte, lengthM)

		_, err = io.ReadFull(p.conn, messageBuf)
		if err != nil {
			log.Printf("Error reading from peer: %s\n", err)
			close(out)
			return
		}

		msgQueue <- Message{
			ID:      messageBuf[0],
			Payload: messageBuf[1:],
		}
	}

}

func (p *Peer) connectPeer(infoHash [20]byte) error {
	handshake := handshakeP{
		ptrLength: byte(19),
		infoHash:  infoHash,
	}

	strHost := p.host.IP.String()
	copy(handshake.protocol[:], "BitTorrent protocol")
	copy(handshake.peerID[:], "-SHOToTorrent-0.1---")

	token := make([]byte, 3)
	rand.Read(token)
	copy(handshake.peerID[17:], token)

	buf := tracker.StructToBuffer(handshake)
	host := p.host.IP.String() + ":" + strconv.Itoa(int(p.host.Port))
	log.Printf("Trying to connect %s...\n", strHost)
	c, err := net.DialTimeout("tcp", host, 20*time.Second)
	if err != nil {
		log.Printf("Cant connect peer (%s)\n", strHost)
		p.host.Status = PEER_DOWN
		return err
	}

	p.conn = c
	log.Printf("connected To Peer (%s)\n", strHost)
	c.Write(buf)

	buffer := make([]byte, 68)
	c.Read(buffer)

	answer := p.unMarshallHandShake(buffer)
	if infoHash != answer.infoHash {
		log.Printf("Invalid Infohash with peer (%s)\n", strHost)
		p.host.Status = PEER_NOINFOHASH
		c.Close()
		return errors.New("Invalid infoHash in handshake with peer")
	}

	log.Printf("HandShake received from Peer: %s - %s\n", strHost, answer.peerID)
	return nil
}

func (p *Peer) newPiece(piecesList *atomicPieces) *StPiece {
	piece := piecesList.findPiece(p.bitfield)
	if piece == nil {
		log.Printf("This peer doesnt have any useful piece")
		p.host.Status = PEER_NOPIECES
		return nil
	}

	p.currentPieceSize = uint32(p.torrent.PieceLength)
	if (uint64((piece.Order + 1) * p.torrent.PieceLength)) > p.torrent.Length {
		p.currentPieceSize = uint32(p.torrent.Length - uint64(piece.Order*p.torrent.PieceLength))
	}
	p.bytesRcvd = 0
	p.bytesReq = 0
	p.currentPieceNum = uint32(piece.Order)
	log.Printf("(%s) current PIECE %d - size: %d\n", p.host.IP.String(), piece.Order, p.currentPieceSize)
	for i := 0; i < 5; i++ {
		p.sendPieceRequest()
	}

	p.status = 3
	return piece
}

func (p *Peer) checkIntegrity(hash [20]byte) error {
	h := sha1.New()
	h.Write(p.currentPieceData[:p.currentPieceSize])
	sha1Piece := h.Sum(nil)
	if !bytes.Equal(sha1Piece, hash[:]) {
		return errors.New("SHA1 Error check")
	}
	return nil
}

func (p *Peer) Start(piecesList *atomicPieces) {

	var msg Message

	for p.host = range p.peersQueue {
		if p.host.Status != PEER_NEW {
			continue
		}
		err := p.connectPeer(p.torrent.InfoHash)
		if err != nil {
			p.peersQueue <- p.host
			continue
		}

		p.status = 0 // waiting for bitfield
		p.bitFieldRecv = false
		errorChan := make(chan struct{})
		msgQueue := make(chan Message, 10)
		var currentPiece *StPiece
		go p.readMessage(msgQueue, errorChan)

		readError := false
		for readError == false {
			select {
			case msg = <-msgQueue:
				err := p.processMessage(msg)
				if err != nil {
					log.Printf("Error processing message: %s\n", err)
					readError = true
				}
				if p.status == 2 {
					currentPiece = p.newPiece(piecesList)
					if currentPiece == nil {
						readError = true
						break
					}
				}
				if p.status == 3 {
					if p.bytesRcvd == p.currentPieceSize {
						log.Printf("Piece %d completed - ", currentPiece.Order)
						err := p.checkIntegrity(currentPiece.Hash)
						if err != nil {
							log.Println(err)
							piecesList.addPiece(*currentPiece)
							break
						}
						log.Printf("Piece %d - valid SHA1\n", currentPiece.Order)

						dataPiece := make([]byte, p.currentPieceSize)
						copy(dataPiece, p.currentPieceData[:p.currentPieceSize])
						p.resultsChan <- StPieceResult{
							Data:  dataPiece,
							Order: currentPiece.Order,
						}

						currentPiece = p.newPiece(piecesList)
						if currentPiece == nil {
							readError = true
							break
						}
					}
				}
			case <-errorChan:
				log.Printf("Error reading from peer...\n")
				if currentPiece != nil {
					piecesList.addPiece(*currentPiece)
				}
				readError = true
			}
		}

	}

}
