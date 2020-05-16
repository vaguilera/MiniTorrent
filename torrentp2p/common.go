package torrentp2p

const (
	PEER_NEW = iota
	PEER_DOWN
	PEER_NOINFOHASH
	PEER_NOPIECES
)

type StPiece struct {
	Hash  [20]byte
	Order int
}

type StPieceResult struct {
	Data  []byte
	Order int
}

type Message struct {
	ID      byte
	Payload []byte
}
