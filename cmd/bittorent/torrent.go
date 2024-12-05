package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
)

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

	index := 0
	decoded, err := decodeBencode(data, &index)
	if err != nil {
		fmt.Printf("Error occurred at position %d\n", index)
		return nil, fmt.Errorf("failed to decode dictionary: %v", err)
	}

	torrent, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected dictionary, got %T", decoded)
	}

	return torrent, nil
}

func calculateSHA1Hash(bencodedData []byte) string {
	hash := sha1.New()
	hash.Write(bencodedData)

	return hex.EncodeToString(hash.Sum(nil))
}

func encodeTorrentInfo(torrentInfo map[string]interface{}) []byte {
	return []byte(encodeDict(torrentInfo))
}

func encodeValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%d:%s", len(v), v)
	case int:
		return fmt.Sprintf("i%d", v) + "e"
	case []byte:
		return fmt.Sprintf("%d:%s", len(v), string(v))
	case []interface{}:
		return encodeArray(v)
	case map[string]interface{}:
		return encodeDict(v)
	default:
		fmt.Println("Could not encode: ", v)
		return ""
	}
}

func encodeArray(array []interface{}) string {
	bencodedArray := "l"
	for _, item := range array {
		bencodedArray += encodeValue(item)
	}
	bencodedArray += "e"
	return bencodedArray
}

func encodeDict(dict map[string]interface{}) string {
	bencodedDict := "d"
	var keys []string
	for key := range dict {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		bencodedDict += fmt.Sprintf("%d:%s", len(key), key)
		bencodedDict += encodeValue(dict[key])
	}
	bencodedDict += "e"
	return bencodedDict
}
