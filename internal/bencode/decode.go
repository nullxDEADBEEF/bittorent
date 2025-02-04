package bencode

import (
	"fmt"
	"sort"
	"strconv"
	"unicode"
	"unicode/utf8"
)

type BencodeDecoder struct {
	data  []byte
	index *int
}

func NewBencodeDecoder(data []byte) *BencodeDecoder {
	index := 0
	return &BencodeDecoder{data, &index}
}

const (
	typeList byte = iota + 'l'
	typeDict      = 'd'
	typeInt       = 'i'
)

const (
	endMarker byte = 'e'
	separator byte = ':'
)

// Bencode (said Bee-encode) is a serialization format used in the Bit torrent protocol
// used to torrent files and in communication between trackers and peers

// Bencode supports 4 data types
// strings
// integers
// arrays
// dictionaries

func (d *BencodeDecoder) Decode() (interface{}, error) {
	if len(d.data) == 0 {
		return nil, fmt.Errorf("empty input data")
	}

	dataType := d.data[*d.index]
	var result interface{}
	var err error

	switch {
	case dataType == typeList:
		result, err = d.decodeList()
	case dataType == typeDict:
		result, err = d.decodeDict()
	case unicode.IsDigit(rune(dataType)):
		result, err = d.decodeString()
	case dataType == typeInt:
		result, err = d.decodeInteger()
	default:
		return nil, fmt.Errorf("invalid data type identifier: %q", dataType)
	}

	if err != nil {
		return nil, err
	}

	return d.convertBytesToStringIfValid(result), nil
}

// strings are encoded as <length>:<content>
func (d *BencodeDecoder) decodeString() (interface{}, error) {
	var firstColonIndex int
	colonFound := false

	for i := *d.index; i < len(d.data); i++ {
		if d.data[i] == ':' {
			firstColonIndex = i
			colonFound = true
			break
		}
	}

	if !colonFound {
		return nil, fmt.Errorf("invalid string, missing colon separator")
	}

	lengthStr := string(d.data[*d.index:firstColonIndex])
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid string length: %w", err)
	}

	contentStart := firstColonIndex + 1
	contentEnd := contentStart + length

	if contentEnd > len(d.data) {
		return nil, fmt.Errorf("invalid string: content length exceeds data length")
	}

	*d.index = contentEnd
	return d.data[contentStart:contentEnd], nil
}

// integers are encoded as i<number>e
// example i52e => 52,   i-52e => -52
func (d *BencodeDecoder) decodeInteger() (interface{}, error) {
	*d.index++
	endIndex := -1

	for i := *d.index; i < len(d.data); i++ {
		if d.data[i] == endMarker {
			endIndex = i
			break
		}
	}

	if endIndex == -1 {
		return nil, fmt.Errorf("invalid integer: missing end marker")
	}

	integer, err := strconv.Atoi(string(d.data[*d.index:endIndex]))
	if err != nil {
		return nil, fmt.Errorf("invalid integer value: %w", err)
	}

	*d.index = endIndex + 1
	return integer, nil
}

// arrays are encoded as l<bencoded_elements>e
//
// lli4eei5ee
// l5:helloi52ee
// lli376e6:orangeee
func (d *BencodeDecoder) decodeList() (interface{}, error) {
	if len(d.data) <= 2 {
		return []interface{}{}, nil
	}

	*d.index++
	var elements []interface{}

	for *d.index < len(d.data) && d.data[*d.index] != endMarker {
		element, err := d.Decode()
		if err != nil {
			return nil, fmt.Errorf("error decoding element: %w", err)
		}
		elements = append(elements, element)
	}

	if *d.index >= len(d.data) {
		return nil, fmt.Errorf("invalid list: missing end marker")
	}

	*d.index++
	return elements, nil
}

// dictionaries are encoded as d<key1><value1>...<keyN><valueN>e
// keys are sorted in lexicographical order and must be strings
//
// {"hello": 52, "foo":"bar"} => d3:foo3:bar5:helloi52ee
// {"inner_dict":{"key1":"value1","key2":42,"list_key":["item1","item2",3]}} => d10:inner_dictd4:key16:value14:key2i42e8:list_keyl5:item15:item2i3eeee
func (d *BencodeDecoder) decodeDict() (interface{}, error) {
	if len(d.data) <= 2 {
		return map[string]interface{}{}, nil
	}

	*d.index++
	dictionary := map[string]interface{}{}

	for *d.index < len(d.data) && d.data[*d.index] != endMarker {
		key, err := d.decodeString()
		if err != nil {
			return nil, fmt.Errorf("error decoding dictionary key: %w", err)
		}

		value, err := d.Decode()
		if err != nil {
			return nil, fmt.Errorf("error decoding value for key %q: %w", key, err)
		}

		keyStr := string(key.([]byte))
		dictionary[keyStr] = d.convertBytesToStringIfValid(value)
	}

	if *d.index >= len(d.data) {
		return nil, fmt.Errorf("invalid dictionary: missing end marker")
	}

	*d.index++
	return d.sortDictionary(dictionary), nil
}

func (d *BencodeDecoder) convertBytesToStringIfValid(value interface{}) interface{} {
	if bytes, ok := value.([]byte); ok && utf8.Valid(bytes) {
		return string(bytes)
	}

	return value
}

func (d *BencodeDecoder) sortDictionary(dictionary map[string]interface{}) map[string]interface{} {
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
