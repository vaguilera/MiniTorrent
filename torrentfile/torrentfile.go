package torrentfile

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	bencode "github.com/jackpal/bencode-go"
)

func (tf *torrentFile) pieceHashes() ([][20]byte, error) {

	buffer := []byte(tf.Info.Pieces)
	lenbuffer := len(buffer)

	if lenbuffer%20 != 0 {
		err := errors.New("Corrupted data in pieces")
		return nil, err
	}

	hashes := make([][20]byte, lenbuffer/20)
	for i := 0; i < len(hashes); i++ {
		copy(hashes[i][:], buffer[i*20:(i+1)*20])
	}
	return hashes, nil

}

func generateInfoHash(info interface{}) ([20]byte, error) {
	buf := bytes.Buffer{}
	err := bencode.Marshal(&buf, info)
	if err != nil {
		return [20]byte{}, err
	}
	hash := sha1.Sum(buf.Bytes())

	return hash, err
}

func (tf *torrentFile) printInfo() {
	log.Printf("Trackers: %v\n", tf.AnnounceList)
	log.Printf("Creation Date: %s\n", time.Unix(int64(tf.CreationDate), 0))
	log.Printf("Comment: %s\n", tf.Comment)
	log.Printf("Created By: %s\n", tf.CreatedBy)
	if tf.Info.Length > 0 {
		log.Printf("File Size: %d\n", tf.Info.Length)
	}
	log.Printf("Filename: %s\n", tf.Info.Name)
	if len(tf.Info.Files) > 0 {
		log.Printf("Files: %v\n", tf.Info.Files)
	}
}

func newTorrent(tf *torrentFile) (*Torrent, error) {

	t := new(Torrent)

	if tf.Info.Length > 0 {
		t.Length = tf.Info.Length
	} else {
		for _, f := range tf.Info.Files {
			t.Length += f.Length
		}
	}
	t.Name = tf.Info.Name
	t.Files = tf.Info.Files

	var err error
	t.PieceLength = tf.Info.PieceLength
	t.PieceHashes, err = tf.pieceHashes()
	if err != nil {
		log.Fatal("Error processing torrent file:", err)
		return nil, err
	}

	var u *url.URL
	var tr tracker

	for i := 0; i < len(tf.AnnounceList); i++ {
		for j := 0; j < len(tf.AnnounceList[i]); j++ {
			u, err = url.Parse(strings.TrimSpace(tf.AnnounceList[i][j]))
			if err != nil {
				log.Fatal("Error processing torrent file:", err)
				return nil, err
			}
			tr.Protocol = u.Scheme
			tr.URL = u.Host

			t.Trackers = append(t.Trackers, tr)
		}
	}
	tf.printInfo()
	log.Printf("Files total length: %d\n\n", t.Length)
	return t, nil

}

// TorrentFromFile creates Torrent entity from .torrent file
func TorrentFromFile(fileName string) (*Torrent, error) {

	file, err := os.Open(fileName)
	defer file.Close()

	if err != nil {
		return nil, err
	}

	fileDecoded, err := bencode.Decode(file)
	if err != nil {
		err = errors.New("Couldn't parse torrent file 1:" + err.Error())
		return nil, err
	}

	mapper, ok := fileDecoded.(map[string]interface{})
	if !ok {
		return nil, errors.New("Couldn't parse torrent file 2")
	}

	infoHash, _ := generateInfoHash(mapper["info"])
	log.Printf("InfoHash: %x\n", infoHash)

	file.Seek(0, 0)
	tfile := torrentFile{}
	err = bencode.Unmarshal(file, &tfile)
	if err != nil {
		err = errors.New("Couldn't parse torrent file 3:" + err.Error())
		return nil, err
	}

	torrent, err := newTorrent(&tfile)
	torrent.InfoHash = infoHash

	return torrent, nil
}
