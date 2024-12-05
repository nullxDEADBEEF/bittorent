package main

import (
	"fmt"
	"sort"
	"strconv"
	"unicode"
	"unicode/utf8"
)

// Bencode (said Bee-encode) is a serialization format used in the Bit torrent protocol
// used to torrent files and in communication between trackers and peers

// Bencode supports 4 data types
// strings
// integers
// arrays
// dictionaries

func decodeBencode(bencodedData []byte, index *int) (interface{}, error) {
	datatypeIdentifer := bencodedData[*index]

	var result interface{}
	var err error

	switch {
	case datatypeIdentifer == 'l':
		result, err = decodeArray(bencodedData, index)
	case datatypeIdentifer == 'd':
		result, err = decodeDictionary(bencodedData, index)
	case unicode.IsDigit(rune(datatypeIdentifer)):
		result, err = decodeString(bencodedData, index)
	case datatypeIdentifer == 'i':
		result, err = decodeInteger(bencodedData, index)
	default:
		return nil, fmt.Errorf("unexpected value %q", bencodedData[*index])
	}

	if err != nil {
		return nil, err
	}

	// Convert []byte to string if it's valid UTF-8
	if valueBytes, ok := result.([]byte); ok {
		if utf8.Valid(valueBytes) {
			return string(valueBytes), nil
		}
	}

	return result, nil
}

// strings are encoded as <length>:<content>
func decodeString(bencodedData []byte, index *int) (interface{}, error) {
	var firstColonIndex int
	colonFound := false

	for i := *index; i < len(bencodedData); i++ {
		if bencodedData[i] == ':' {
			firstColonIndex = i
			colonFound = true
			break
		}
	}

	if !colonFound {
		return nil, fmt.Errorf("invalid string, missing colon separator")
	}

	lengthStr := string(bencodedData[*index:firstColonIndex])
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid length: %v", err)
	}

	if firstColonIndex+1+length > len(bencodedData) {
		return nil, fmt.Errorf("invalid length: expected %d bytes after colon, but only %d bytes remain",
			length, len(bencodedData)-(firstColonIndex+1))
	}

	*index = firstColonIndex + 1 + length
	return bencodedData[firstColonIndex+1 : firstColonIndex+1+length], nil
}

// integers are encoded as i<number>e
// example i52e => 52,   i-52e => -52
func decodeInteger(bencodedData []byte, index *int) (interface{}, error) {
	var endIndex int

	*index += 1
	for i := *index; i < len(bencodedData); i++ {
		if bencodedData[i] == 'e' {
			endIndex = i
			break
		}
	}

	integer, err := strconv.Atoi(string(bencodedData[*index:endIndex]))
	if err != nil {
		return nil, fmt.Errorf("invalid integer value: %v", err)
	}

	*index += (endIndex - *index) + 1

	return integer, nil
}

// arrays are encoded as l<bencoded_elements>e
//
// lli4eei5ee
// l5:helloi52ee
// lli376e6:orangeee
func decodeArray(bencodedData []byte, index *int) (interface{}, error) {
	if len(bencodedData) <= 2 {
		return []interface{}{}, nil
	}

	*index += 1
	elements := make([]interface{}, 0)

	for {
		result, err := decodeBencode(bencodedData, index)
		if err != nil {
			return nil, err
		}

		if valueBytes, ok := result.([]byte); ok {
			if utf8.Valid(valueBytes) {
				result = string(valueBytes)
			}
		}

		elements = append(elements, result)

		if bencodedData[*index] == 'e' {
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
func decodeDictionary(bencodedData []byte, index *int) (interface{}, error) {
	if len(bencodedData) <= 2 {
		return map[string]interface{}{}, nil
	}

	*index += 1
	dictionary := map[string]interface{}{}

	for {
		key, err := decodeString(bencodedData, index)
		if err != nil {
			return nil, fmt.Errorf("error decoding key: %v", err)
		}

		value, err := decodeBencode(bencodedData, index)
		if err != nil {
			return nil, fmt.Errorf("error decoding value for key %q: %v", key, err)
		}

		keyStr := string(key.([]byte))

		if valueBytes, ok := value.([]byte); ok {
			if utf8.Valid(valueBytes) {
				dictionary[keyStr] = string(valueBytes)
			} else {
				dictionary[keyStr] = valueBytes
			}
		} else {
			dictionary[keyStr] = value
		}

		if bencodedData[*index] == 'e' {
			*index += 1
			break
		}
	}

	return sortMapLexicographically(dictionary), nil
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
