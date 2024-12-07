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
		handleDecode(os.Args[2])
	case "info":
		handleInfo(os.Args[2])
	case "peers":
		handlePeers(os.Args[2])
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}

func handleDecode(bencodedValue string) {
	decoder := NewBencodeDecoder([]byte(bencodedValue))
	decoded, err := decoder.Decode()
	if err != nil {
		fmt.Println(err)
		return
	}
	jsonOutput, _ := json.Marshal(decoded)
	fmt.Println(string(jsonOutput))
}

func handleInfo(torrentPath string) {
	torrent, err := parseTorrentFile(torrentPath)
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
}

func handlePeers(torrentPath string) {
	torrent, err := parseTorrentFile(torrentPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	encoder := NewTorrentEncoder()
	torrentInfo := torrent["info"].(map[string]interface{})
	bencodedInfo := encoder.encodeTorrentInfo(torrentInfo)
	infoHash := encoder.CalculateSHA1Hash(bencodedInfo)

	peers, err := getPeers(torrent, torrentInfo, infoHash)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, peer := range peers {
		fmt.Println(peer)
	}
}

func getPeers(torrent map[string]interface{}, torrentInfo map[string]interface{}, infoHash string) ([]string, error) {
	encodedInfoHash := encodeInfoHash(infoHash)

	req, err := http.NewRequest("GET", torrent["announce"].(string), nil)
	if err != nil {
		return nil, err
	}

	rawQuery := fmt.Sprintf("info_hash=%s&peer_id=99999999999999999999&port=6881&uploaded=0&downloaded=0&left=%d&compact=1",
		encodedInfoHash,
		torrentInfo["length"])
	req.URL.RawQuery = rawQuery

	resp, err := http.Get(req.URL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	decoder := NewBencodeDecoder(body)
	decoded, err := decoder.Decode()
	if err != nil {
		return nil, err
	}

	return parsePeers(decoded.(map[string]interface{})["peers"].([]byte)), nil
}

func encodeInfoHash(infoHash string) string {
	var encodedInfoHash string

	// Convert hex string to bytes and encode each byte
	for i := 0; i < len(infoHash); i += 2 {
		var byteVal byte
		// Convert each pair of hex chars to a byte
		fmt.Sscanf(infoHash[i:i+2], "%02x", &byteVal)
		encodedInfoHash += fmt.Sprintf("%%%02x", byteVal)
	}
	return encodedInfoHash
}

func parsePeers(peersBytes []byte) []string {
	peers := make([]string, 0)
	ipWithPortInBytes := 6
	for i := 0; i < len(peersBytes); i += ipWithPortInBytes {
		ip := fmt.Sprintf("%d.%d.%d.%d", peersBytes[i], peersBytes[i+1], peersBytes[i+2], peersBytes[i+3])
		port := binary.BigEndian.Uint16(peersBytes[i+4 : i+6])
		peers = append(peers, fmt.Sprintf("%s:%d", ip, port))
	}
	return peers
}
