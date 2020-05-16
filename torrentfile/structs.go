package torrentfile

type TorrentMultiFileInfo struct {
	Length uint64   `bencode:"length"`
	Path   []string `bencode:"path"`
	MD5sum string   `bencode:"md5sum"`
}

type torrentFileInfo struct {
	Pieces      string                 `bencode:"pieces"`
	PieceLength int                    `bencode:"piece length"`
	Length      uint64                 `bencode:"length"`
	Name        string                 `bencode:"name"`
	Files       []TorrentMultiFileInfo `bencode:"files"`
}

type torrentFile struct {
	Announce     string     `bencode:"announce"`
	AnnounceList [][]string `bencode:"announce-list"`
	CreationDate int        `bencode:"creation date"`
	Info         torrentFileInfo
	Comment      string `bencode:"comment"`
	CreatedBy    string `bencode:"created by"`
	RawInfo      string
}

type tracker struct {
	URL      string
	Protocol string
}

// Torrent Represents a torrent entity
type Torrent struct {
	Trackers    []tracker
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      uint64
	Name        string
	Files       []TorrentMultiFileInfo
}
