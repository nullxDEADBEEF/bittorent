package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	case "peers":
		torrent, err := parseTorrentFile(os.Args[2])
		if err != nil {
			fmt.Println(err)
			return
		}

		encoder := NewTorrentEncoder()
		torrentInfo := torrent["info"].(map[string]interface{})
		bencodedInfo := encoder.encodeTorrentInfo(torrentInfo)
		infoHash := encoder.CalculateSHA1Hash(bencodedInfo)

		req, err := http.NewRequest("GET", torrent["announce"].(string), nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Convert hex string to bytes and encode each byte
		var encodedInfoHash string
		for i := 0; i < len(infoHash); i += 2 {
			// Convert each pair of hex chars to a byte
			byteVal := byte(0)
			fmt.Sscanf(infoHash[i:i+2], "%02x", &byteVal)
			encodedInfoHash += fmt.Sprintf("%%%02x", byteVal)
		}

		rawQuery := fmt.Sprintf("info_hash=%s&peer_id=99999999999999999999&port=6881&uploaded=0&downloaded=0&left=%d&compact=1",
			encodedInfoHash,
			torrentInfo["length"])

		req.URL.RawQuery = rawQuery

		resp, err := http.Get(req.URL.String())
		if err != nil {
			fmt.Println(err)
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		resp.Body.Close()

		decoder := NewBencodeDecoder(body)
		decoded, err := decoder.Decode()
		if err != nil {
			fmt.Println(err)
			return
		}

		peersBytes := decoded.(map[string]interface{})["peers"].([]byte)
		peers := make([]string, 0)
		ipWithPortInBytes := 6
		for i := 0; i < len(peersBytes); i += ipWithPortInBytes {
			ip := fmt.Sprintf("%d.%d.%d.%d", peersBytes[i], peersBytes[i+1], peersBytes[i+2], peersBytes[i+3])
			port := binary.BigEndian.Uint16(peersBytes[i+4 : i+6])
			peers = append(peers, fmt.Sprintf("%s:%d", ip, port))
		}

		for _, peer := range peers {
			fmt.Println(peer)
		}

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
