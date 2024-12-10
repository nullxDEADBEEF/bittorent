package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
)

const (
	dictStart = 'd'
	dictEnd   = 'e'
	listStart = 'l'
	listEnd   = 'e'
	intStart  = 'i'
	intEnd    = 'e'
	delimiter = ':'
)

type TorrentEncoder struct{}

func NewTorrentEncoder() *TorrentEncoder {
	return &TorrentEncoder{}
}

// torrent file(also known as metainfo file) contains bencoded dictionary with the following keys and values:
// announce => URL to a "tracker", a central server that keeps track of peers participating in the sharing of a torrent
// info, dictionary with keys
//   - length: size of the file in bytes, for single-file torrents
//   - name: suggested name to save the file / directory as
//   - piece length: number of bytes in each piece
//   - pieces: concatenated SHA-1 hashes of each piece
//
// NOTE: info dictionary is slightly different for multi-file torrents
func parseTorrentFile(filepath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	decoder := NewBencodeDecoder(data)
	decoded, err := decoder.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode dictionary: %v", err)
	}

	torrent, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected dictionary, got %T", decoded)
	}

	return torrent, nil
}

func (e *TorrentEncoder) GetTorrentPieceHashes(bencodedPieces []byte) []string {
	pieceHashes := make([]string, 0)
	hashSizeInBytes := 20

	for i := 0; i < len(bencodedPieces); i += hashSizeInBytes {
		pieceHashes = append(pieceHashes, hex.EncodeToString(bencodedPieces[i:i+hashSizeInBytes]))
	}

	return pieceHashes
}

func (e *TorrentEncoder) PrintPieceHashes(pieceHashes []string) {
	for _, hash := range pieceHashes {
		fmt.Println(hash)
	}
}

func (e *TorrentEncoder) CalculateSHA1Hash(bencodedInfo []byte) string {
	hash := sha1.New()
	hash.Write(bencodedInfo)

	return hex.EncodeToString(hash.Sum(nil))
}

func (e *TorrentEncoder) encodeTorrentInfo(torrentInfo map[string]interface{}) []byte {
	return []byte(e.encodeDict(torrentInfo))
}

func (e *TorrentEncoder) encodeValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%d:%s", len(v), v)
	case int:
		return fmt.Sprintf("i%d", v) + "e"
	case []byte:
		return fmt.Sprintf("%d:%s", len(v), string(v))
	case []interface{}:
		return e.encodeArray(v)
	case map[string]interface{}:
		return e.encodeDict(v)
	default:
		fmt.Println("Could not encode: ", v)
		return ""
	}
}

func (e *TorrentEncoder) encodeArray(array []interface{}) string {
	bencodedArray := "l"
	for _, item := range array {
		bencodedArray += e.encodeValue(item)
	}
	bencodedArray += "e"
	return bencodedArray
}

func (e *TorrentEncoder) encodeDict(dict map[string]interface{}) string {
	result := string(dictStart)

	keys := make([]string, 0, len(dict))
	for key := range dict {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		result += fmt.Sprintf("%d%c%s", len(key), delimiter, key)
		result += e.encodeValue(dict[key])
	}

	return result + string(dictEnd)
}
