package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	command := os.Args[1]

	switch command {
	case "decode":
		bencodedValue := os.Args[2]

		decoder := NewBencodeDecoder([]byte(bencodedValue))
		decoded, err := decoder.Decode()
		if err != nil {
			fmt.Println(err)
			return
		}
		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	case "info":
		torrent, err := parseTorrentFile(os.Args[2])
		if err != nil {
			fmt.Println(err)
			return
		}

		encoder := NewTorrentEncoder()
		bencodedInfo := encoder.encodeTorrentInfo(torrent["info"].(map[string]interface{}))
		infoHash := calculateSHA1Hash(bencodedInfo)

		torrentInfo := torrent["info"].(map[string]interface{})

		fmt.Printf("Tracker URL: %s\nLength: %d\nInfo Hash: %s\nPiece Length: %d\n",
			torrent["announce"],
			torrentInfo["length"],
			infoHash,
			torrentInfo["piece length"])
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
