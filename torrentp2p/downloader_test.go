package torrentp2p

import (
	"testing"
)

func Test_findPiece(t *testing.T) {

	var pieces []StPiece
	peerPieces := []byte{0, 1, 0, 0}
	for i := 0; i < 4; i++ {
		pieces = append(pieces, StPiece{
			Hash:  [20]byte{},
			Order: i,
		})
	}

	APieces := atomicPieces{
		pieces: pieces,
	}

	mypiece := APieces.findPiece(peerPieces)

	if mypiece.Order != 1 {
		t.Errorf("Expected piece 1, got %d", mypiece.Order)
	}
	if len(APieces.pieces) != 3 {
		t.Errorf("Expected 3 pieces left in list, got %d", len(APieces.pieces))
	}
	if APieces.pieces[0].Order != 0 ||
		APieces.pieces[1].Order != 3 ||
		APieces.pieces[2].Order != 2 {
		t.Errorf("Unexpected piece order in remaining list: %v", APieces.pieces)
	}
}
