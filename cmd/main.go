package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/vaguilera/MiniTorrent/torrentfile"
	"github.com/vaguilera/MiniTorrent/torrentp2p"
)

func printHelp() {
	fmt.Printf("MiniTorrent client V1.0\nUsage:\n\tminitorrent -W=<NumOfWorkers> <torrentfile>\n")
	flag.PrintDefaults()
}

type fichero struct {
	name string
	size uint32
}

func main() {

	log.SetFlags(0)

	workers := flag.Int("w", 4, "Number of workers")
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		printHelp()
		os.Exit(2)
	}

	torrentFile, err := torrentfile.TorrentFromFile(args[0])

	if err != nil {
		log.Fatalf("Error while opening file: %s", err)
	}

	downloader := new(torrentp2p.Downloader)
	downloader.Run(torrentFile, *workers)

}
