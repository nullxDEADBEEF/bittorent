package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
)

// Bencode (said Bee-encode) is a serialization format used in the Bit torrent protocol
// used to torrent files and in communication between trackers and peers

// Bencode supports 4 data types
// strings
// integers
// arrays
// dictionaries

func decodeBencode(bencodedString string, index *int) (interface{}, error) {
	if len(bencodedString) <= 2 {
		return []string{}, nil
	}

	datatypeIdentifer := rune(bencodedString[*index])

	switch {
	case datatypeIdentifer == 'l':
		return decodeArray(bencodedString, index)
	case unicode.IsDigit(datatypeIdentifer):
		return decodeString(bencodedString, index)
	case datatypeIdentifer == 'i':
		return decodeInteger(bencodedString, index)
	default:
		return nil, fmt.Errorf("unexpected value %q", bencodedString[*index])
	}
}

// strings are encoded as <length>:<content>
func decodeString(bencodedString string, index *int) (interface{}, error) {
	var firstColonIndex int

	for i := *index; i < len(bencodedString); i++ {
		if bencodedString[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	lengthStr := bencodedString[*index:firstColonIndex]

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", err
	}

	*index += length + 2
	return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], nil
}

// integers are encoded as i<number>e
// example i52e => 52,   i-52e => -52
func decodeInteger(bencodedString string, index *int) (interface{}, error) {
	var endIndex int

	*index += 1
	for i := *index; i < len(bencodedString); i++ {
		if bencodedString[i] == 'e' {
			endIndex = i
			break
		}
	}

	integer, err := strconv.Atoi(bencodedString[*index:endIndex])
	if err != nil {
		return "", err
	}

	*index += (endIndex - *index) + 1

	return integer, nil
}

// arrays are encoded as l<bencoded_elements>e
//
// lli4eei5ee
// l5:helloi52ee
// lli376e6:orangeee
func decodeArray(bencodedString string, index *int) (interface{}, error) {
	*index += 1
	elements := make([]interface{}, 0)

	for {
		result, err := decodeBencode(bencodedString, index)
		if err != nil {
			fmt.Println(err)
		}
		elements = append(elements, result)

		if bencodedString[*index] == 'e' {
			*index += 1
			break
		}
	}

	return elements, nil
}

func main() {
	command := os.Args[1]

	if command == "decode" {
		bencodedValue := os.Args[2]

		index := 0
		decoded, err := decodeBencode(bencodedValue, &index)
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
