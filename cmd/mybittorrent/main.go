package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// Bencode (said Bee-encode) is a serialization format used in the Bit torrent protocol
// used to torrent files and in communication between trackers and peers

// Bencode supports 4 data types
// strings
// integers
// arrays
// dictionaries

// strings are encoded as <length>:<content>
func decodeBencode(bencodedString string) (interface{}, error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		return decodeString(bencodedString)
	} else if rune(bencodedString[0]) == 'i' {
		return decodeInteger(bencodedString)
	}

	return "", nil
}

func decodeString(bencodedString string) (interface{}, error) {
	var firstColonIndex int

	for i := 0; i < len(bencodedString); i++ {
		if bencodedString[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	lengthStr := bencodedString[:firstColonIndex]

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", err
	}

	return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], nil
}

// integers are encoded as i<number>e
// example i52e => 52,   i-52e => -52
func decodeInteger(bencodedString string) (interface{}, error) {
	endIndex := strings.Index(bencodedString, "e")

	integer, err := strconv.Atoi(bencodedString[1:endIndex])
	if err != nil {
		return "", err
	}

	return integer, nil
}

func main() {
	command := os.Args[1]

	if command == "decode" {
		bencodedValue := os.Args[2]

		decoded, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
