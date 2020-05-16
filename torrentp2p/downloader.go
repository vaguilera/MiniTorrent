package torrentp2p

import (
	"log"
	"net"
	"os"
	"sync"

	"github.com/vaguilera/MiniTorrent/torrentfile"
	"github.com/vaguilera/MiniTorrent/tracker"
)

type atomicPieces struct {
	mu     sync.Mutex
	pieces []StPiece
}

type Downloader struct {
	peers       []tracker.Peer
	peersQueue  chan tracker.Peer
	ownedPieces int
}

func (p *atomicPieces) findPiece(peerPieces []byte) *StPiece {
	p.mu.Lock()         // Wait for the lock to be free and then take it.
	defer p.mu.Unlock() // Release the lock.
	for i := range p.pieces {
		if peerPieces[p.pieces[i].Order] == 1 {
			cPiece := p.pieces[i]
			if len(p.pieces) > 1 { // we left always 1 piece into the queue to prevent 1 peer blocks the download
				p.pieces[i] = p.pieces[len(p.pieces)-1]
				p.pieces = p.pieces[:len(p.pieces)-1]
			}
			return &cPiece
		}
	}
	return nil
}

func (p *atomicPieces) addPiece(piece StPiece) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.pieces = append(p.pieces, piece)
}

func (down *Downloader) peerExists(peer tracker.Peer) bool {
	for _, exPeers := range down.peers {
		if (peer.IP.String() == exPeers.IP.String()) && (peer.Port == exPeers.Port) {
			return true
		}
	}
	return false
}

func (down *Downloader) getPeers(host string, infoHash [20]byte, torrentLength uint64) error {
	tracker := &tracker.UDPTracker{
		Host:     host,
		InfoHash: infoHash,
		Length:   torrentLength,
	}

	err := tracker.Connect()
	if err != nil {
		log.Printf("%s ...KO\n", host)
		return err
	}

	peers, err := tracker.Announce()
	if err != nil {
		log.Printf("%s ...KO (Error announcing)\n", host)
		return err
	}

	for _, p := range peers {
		if !down.peerExists(p) {
			down.peers = append(down.peers, p)
		}
	}

	return nil
}

func (down *Downloader) scrapTrackers(torrent *torrentfile.Torrent) {
	for _, t := range torrent.Trackers {
		if t.Protocol == "udp" {
			log.Printf("Retrieving peers from %s\n", t.URL)
			err := down.getPeers(t.URL, torrent.InfoHash, torrent.Length)
			if err == nil {
				break
			}
		}

	}
}

func (down *Downloader) initPiecesList(numPieces int, torrent *torrentfile.Torrent, APieces *atomicPieces) {
	var pieces []StPiece

	for i := 0; i < numPieces; i++ {
		pieces = append(pieces, StPiece{
			Hash:  torrent.PieceHashes[i],
			Order: i,
		})
	}

	APieces.pieces = pieces
}

func (down *Downloader) initPeersQueue() {
	down.peersQueue = make(chan tracker.Peer, len(down.peers))

	for _, cPeer := range down.peers {
		cPeer.Status = PEER_NEW
		down.peersQueue <- cPeer
	}

}

func (down *Downloader) Run(torrent *torrentfile.Torrent, numWorkers int) {

	log.Printf("Number of workers: %d\n", numWorkers)
	piecesList := atomicPieces{}
	numPieces := len(torrent.PieceHashes)
	filewriter := fileWriter{}

	filewriter.CreateFiles(torrent.Files)
	defer filewriter.closeFiles()

	if os.Getenv("TEST_LOCAL_CLIENT") == "true" {
		log.Printf("- DEBUG: Using local connection. Not scrapping peers.")
		down.peers = append(down.peers, tracker.Peer{
			IP:     net.ParseIP("127.0.0.1"),
			Port:   25771,
			Status: PEER_NEW,
		})
	} else {
		down.scrapTrackers(torrent)
	}

	down.initPiecesList(numPieces, torrent, &piecesList)
	down.initPeersQueue()

	resultsChan := make(chan StPieceResult, 1)

	for i := 0; i < numWorkers; i++ {
		worker := NewPeer(
			torrent,
			down.peersQueue,
			resultsChan,
		)
		go worker.Start(&piecesList)
	}

	for {
		res := <-resultsChan
		filewriter.writeData(res.Data, uint64(res.Order*torrent.PieceLength))
		down.ownedPieces++
		if down.ownedPieces == numPieces {
			break
		}
	}

	log.Println("File(s) downloaded")
}
