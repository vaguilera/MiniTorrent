package torrentp2p

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaguilera/MiniTorrent/torrentfile"
)

func create(p string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0600); err != nil {
		return nil, err
	}
	return os.OpenFile(p, os.O_WRONLY|os.O_CREATE, 0600)
}

type fileData struct {
	file   *os.File
	length uint64
}

type fileWriter struct {
	files     []fileData
	multifile bool
}

func (fw *fileWriter) closeFiles() {
	for _, file := range fw.files {
		file.file.Close()
	}
}

func (fw *fileWriter) writeData(data []byte, offset uint64) error {
	if !fw.multifile {
		_, err := fw.files[0].file.WriteAt(data, int64(offset))
		return err
	}
	cOffset := uint64(0)
	dataOff := uint64(0)
	i := 0
	dataLength := uint64(len(data))
	for dataOff < dataLength {
		cOffset += fw.files[i].length
		if cOffset > offset {
			relative := offset - (cOffset - fw.files[i].length)
			relData := fw.files[i].length - relative
			if relData > dataLength-dataOff {
				relData = dataLength - dataOff
			}
			_, err := fw.files[i].file.WriteAt(data[dataOff:dataOff+relData], int64(relative))
			if err != nil {
				return err
			}
			dataOff += relData
			offset += relData
		}
		i++
	}

	return nil
}

func (fw *fileWriter) CreateFiles(files []torrentfile.TorrentMultiFileInfo) error {
	if _, err := os.Stat("download"); os.IsNotExist(err) {
		err = os.Mkdir("download", 0600)
		if err != nil {
			return err
		}
	}

	var filePath string
	for _, file := range files {
		if len(file.Path) > 1 {
			lastItem := len(file.Path) - 1
			pathString := strings.Join(file.Path[:lastItem], "/")
			filePath = pathString + "/" + file.Path[lastItem]
		} else {
			filePath = file.Path[0]
		}
		cfile, err := create("download/" + filePath)
		if err != nil {
			log.Printf("error creating : %s", filePath)
			return err
		}
		fw.files = append(fw.files, fileData{file: cfile, length: file.Length})
	}

	if len(fw.files) > 1 {
		fw.multifile = true
	}
	return nil
}
