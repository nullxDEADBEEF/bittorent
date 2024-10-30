package main

import (
	"fmt"
	"sort"
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
	datatypeIdentifer := rune(bencodedString[*index])

	switch {
	case datatypeIdentifer == 'l':
		return decodeArray(bencodedString, index)
	case datatypeIdentifer == 'd':
		return decodeDictionary(bencodedString, index)
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

	numDigits := -1
	for range lengthStr {
		numDigits++
	}

	*index += length + numDigits + 2
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
	if len(bencodedString) <= 2 {
		return []string{}, nil
	}

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

// dictionaries are encoded as d<key1><value1>...<keyN><valueN>e
// keys are sorted in lexicographical order and must be strings
//
// {"hello": 52, "foo":"bar"} => d3:foo3:bar5:helloi52ee
// {"inner_dict":{"key1":"value1","key2":42,"list_key":["item1","item2",3]}} => d10:inner_dictd4:key16:value14:key2i42e8:list_keyl5:item15:item2i3eeee
func decodeDictionary(bencodedString string, index *int) (interface{}, error) {
	if len(bencodedString) <= 2 {
		return map[string]string{}, nil
	}

	*index += 1
	dictionary := map[string]interface{}{}

	currentKey := ""
	localIndex := 0

	for {
		result, err := decodeBencode(bencodedString, index)
		if err != nil {
			fmt.Println(err)
		}

		if localIndex%2 == 0 {
			currentKey = result.(string)
			localIndex++
		} else {
			dictionary[currentKey] = result
			localIndex++
		}

		if bencodedString[*index] == 'e' {
			*index += 1
			break
		}
	}

	dictionary = sortMapLexicographically(dictionary)
	return dictionary, nil
}

func sortMapLexicographically(dictionary map[string]interface{}) map[string]interface{} {
	sortedMap := make(map[string]interface{}, len(dictionary))
	keys := make([]string, 0, len(dictionary))
	for k := range dictionary {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		sortedMap[k] = dictionary[k]
	}

	return sortedMap
}
