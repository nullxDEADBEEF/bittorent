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
		torrentInfo := torrent["info"].(map[string]interface{})
		bencodedInfo := encoder.encodeTorrentInfo(torrentInfo)
		infoHash := encoder.CalculateSHA1Hash(bencodedInfo)
		pieceHashes := encoder.GetTorrentPieceHashes(torrentInfo["pieces"].([]byte))

		fmt.Printf("Tracker URL: %s\nLength: %d\nInfo Hash: %s\nPiece Length: %d\nPiece Hashes:\n",
			torrent["announce"],
			torrentInfo["length"],
			infoHash,
			torrentInfo["piece length"])
		encoder.PrintPieceHashes(pieceHashes)
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
