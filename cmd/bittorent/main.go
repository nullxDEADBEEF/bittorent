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

		index := 0
		decoded, err := decodeBencode(bencodedValue, &index)
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

		bencodedInfo := encodeTorrentInfo(torrent["info"].(map[string]interface{}))
		infoHash := calculateSHA1Hash(bencodedInfo)

		fmt.Printf("Tracker URL: %s\nLength: %d\nInfo Hash: %s\n",
			torrent["announce"],
			torrent["info"].(map[string]interface{})["length"],
			infoHash)
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
