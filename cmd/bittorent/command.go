package main

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
)

const (
	BITFIELD_ID         byte = 5
	INTERESTED_ID       byte = 2
	UNCHOKE_ID          byte = 1
	REQUEST_ID          byte = 6
	PIECE_ID            byte = 7
	PAYLOAD_BYTES       byte = 17
	LengthIndexStart    byte = 0
	LengthIndexEnd      byte = 4
	PieceIndexStart     byte = 5
	PieceIndexEnd       byte = 9
	OffsetIndexStart    byte = 9
	OffsetIndexEnd      byte = 13
	BlocksizeIndexStart byte = 13
	BlocksizeIndexEnd   byte = 17
	PieceDataStart      byte = 9
)

func handleDecode(bencodedValue string) string {
	decoder := NewBencodeDecoder([]byte(bencodedValue))
	decoded, err := decoder.Decode()
	if err != nil {
		fmt.Println(err)
		return ""
	}
	jsonOutput, _ := json.Marshal(decoded)

	return string(jsonOutput)
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

func handlePeers(torrentPath string) []string {
	torrent, err := parseTorrentFile(torrentPath)
	if err != nil {
		fmt.Println(err)
		return []string{}
	}

	encoder := NewTorrentEncoder()
	torrentInfo := torrent["info"].(map[string]interface{})
	bencodedInfo := encoder.encodeTorrentInfo(torrentInfo)
	infoHash := encoder.CalculateSHA1Hash(bencodedInfo)

	peers, err := getPeers(torrent, torrentInfo, infoHash)
	if err != nil {
		fmt.Println(err)
		return []string{}
	}

	return peers
}

func handleHandshake(torrentPath string, peerIP string) (net.Conn, string) {
	conn, err := net.Dial("tcp", peerIP)
	if err != nil {
		fmt.Println(err)
		return nil, ""
	}

	torrent, err := parseTorrentFile(torrentPath)
	if err != nil {
		fmt.Println(err)
		return nil, ""
	}

	torrentInfo := torrent["info"].(map[string]interface{})
	encoder := NewTorrentEncoder()
	bencodedInfo := encoder.encodeTorrentInfo(torrentInfo)
	infoHash := encoder.CalculateSHA1Hash(bencodedInfo)

	handshake := make([]byte, 0)
	handshake = append(handshake, byte(19))
	handshake = append(handshake, []byte("BitTorrent protocol")...)
	handshake = append(handshake, make([]byte, 8)...)

	infoHashBytes, err := hex.DecodeString(infoHash)
	if err != nil {
		fmt.Println(err)
		return nil, ""
	}
	handshake = append(handshake, infoHashBytes...)

	handshake = append(handshake, generatePeerID()...)

	conn.Write(handshake)

	handshakeLength := 68
	response := make([]byte, handshakeLength)
	_, err = conn.Read(response)
	if err != nil {
		fmt.Println(err)
		return nil, ""
	}

	peerIDInHandshake := response[48:]

	return conn, hex.EncodeToString(peerIDInHandshake)
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

func generatePeerID() []byte {
	peerID := make([]byte, 20)
	_, err := rand.Read(peerID)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return peerID
}

/*
peer messages consists of:
- message length prefix (4 bytes)
- message id (1 byte)
- payload (variable size)

// once the handshake is complete we need to exchange the following messages
- wait for _bitfield_ message from peer to indicate which pieces it has
  - message id for this type is 5
  - payload ()

- send _interested_ message
  - id for this message type is 2
  - empty payload

- wait till receiving _unchoke_ message
  - id for this message type is 1
  - empty payload

// Break the pieces into blocks of 16 kiB (16 * 1024 bytes) and send _request_ message for each block
  - id for this message type is 6
  - payload consists of:
  - index: zero-based index
  - begin: zero-based byte offset within the piece
    0 for first block, 2^14 for second, 2 * 2^14 for third.....
  - length: the length of the block in bytes
    this will be 2^14 for all blocks except the last one.
    the last one will be 2^14 bytes or lower, this will be caculated using the piece length

wait till receiving _piece_ message for each block requested
  - id for this message type is 7
  - payload consists of:
    index: zero-based piece index
    begin: zero-based byte offset within the piece
    block: the data for the piece, usually 2^14 bytes long

After combining blocks into pieces, we have to check the integrity of each piece
by comparing the hash with the piece hash found in the torrent file
*/
func downloadPiece(torrentPath string, pieceIndex int) []byte {
	torrent, err := parseTorrentFile(torrentPath)
	if err != nil {
		log.Printf("Failed to parse torrent: %v", err)
	}

	encoder := NewTorrentEncoder()
	torrentInfo := torrent["info"].(map[string]interface{})
	pieceHashes := encoder.GetTorrentPieceHashes(torrentInfo["pieces"].([]byte))
	standardPieceLength := torrentInfo["piece length"].(int)
	fileLength := torrentInfo["length"].(int)

	// ceiling division
	numPieces := (fileLength + standardPieceLength - 1) / standardPieceLength

	// calcualte piece length for this piece
	pieceLength := standardPieceLength
	if pieceIndex == numPieces-1 {
		remainingLength := fileLength - (pieceIndex * standardPieceLength)
		if remainingLength > 0 {
			pieceLength = remainingLength
		}
	}

	peers := handlePeers(torrentPath)
	if len(peers) < 1 {
		fmt.Println("Could not find any peers")
		return nil
	}

	fmt.Println("STARTING HANDSHAKE")

	conn, _ := handleHandshake(torrentPath, peers[0])
	defer conn.Close()

	fmt.Println("HANDSHAKE COMPLETE")

	reader := bufio.NewReader(conn)

	blockOffset := 0
	pieceData := make([]byte, 0)
	atLastBlock := false

	for {
		lengthBuf := make([]byte, 4)
		if _, err := io.ReadFull(reader, lengthBuf); err != nil {
			log.Printf("Failed to read length: %v", err)
			return nil
		}
		messageLength := binary.BigEndian.Uint32(lengthBuf)

		messageBuffer := make([]byte, messageLength)
		if _, err := io.ReadFull(reader, messageBuffer); err != nil {
			log.Printf("Failed to read message: %v", err)
			return nil
		}

		messageID := messageBuffer[0]

		log.Printf("Received message with data length: %d, ID: %d", messageLength-9, messageID)

		switch messageID {
		case BITFIELD_ID:
			msg := []byte{0, 0, 0, 1, INTERESTED_ID}
			if _, err := conn.Write(msg); err != nil {
				log.Printf("Error sending INTERESTED message: %v", err)
				return nil
			}
		case UNCHOKE_ID:
			blockSize := 1 << 14
			if blockSize > pieceLength {
				blockSize = pieceLength
			}
			if err := sendBlockRequest(conn, pieceIndex, blockOffset, blockSize); err != nil {
				log.Printf("Error sending initial block request: %v", err)
				return nil
			}

		case PIECE_ID:
			blockData := messageBuffer[PieceDataStart:]
			pieceData = append(pieceData, blockData...)

			if len(pieceData) == pieceLength {
				atLastBlock = true
			}

			blockOffset += len(blockData)
			if blockOffset < pieceLength {
				blockSize := 1 << 14

				if blockOffset+blockSize > pieceLength {
					blockSize = pieceLength - blockOffset
				}

				if err := sendBlockRequest(conn, pieceIndex, blockOffset, blockSize); err != nil {
					log.Printf("Error sending next block request: %v", err)
					return nil
				}
			}
		default:
			log.Println("Unexpected peer message")
			continue
		}

		if atLastBlock {
			break
		}
	}

	expectedHash := pieceHashes[pieceIndex]
	receivedHash := encoder.CalculateSHA1Hash(pieceData)
	if expectedHash != receivedHash {
		log.Println("Hash from torrent", pieceHashes[pieceIndex])
		log.Println("Hash received from fetched piece", receivedHash[pieceIndex])
	} else {
		log.Println("Hashes match :o!")
	}

	return pieceData
}

func download(torrentPath string) []byte {
	torrent, err := parseTorrentFile(torrentPath)
	if err != nil {
		log.Printf("Failed to parse torrent: %v", err)
	}

	torrentInfo := torrent["info"].(map[string]interface{})
	fileLength := torrentInfo["length"].(int)
	standardPieceLength := torrentInfo["piece length"].(int)
	numPieces := (fileLength + standardPieceLength - 1) / standardPieceLength

	piecesChan := make(chan struct {
		index int
		data  []byte
	}, numPieces)

	// WaitGroup to wait for all downloads to finish
	var wg sync.WaitGroup

	maxConcurrent := 5
	semaphore := make(chan int, maxConcurrent)

	for i := 0; i < numPieces; i++ {
		wg.Add(1)
		go func(pieceIndex int) {
			defer wg.Done()

			semaphore <- 1
			defer func() { <-semaphore }()

			pieceData := downloadPiece(torrentPath, pieceIndex)
			piecesChan <- struct {
				index int
				data  []byte
			}{pieceIndex, pieceData}
		}(i)
	}

	// close pieces channel when all downloads are done
	go func() {
		wg.Wait()
		close(piecesChan)
	}()

	pieces := make([][]byte, numPieces)
	for piece := range piecesChan {
		pieces[piece.index] = piece.data
	}

	fileData := make([]byte, 0, fileLength)
	for i := 0; i < numPieces; i++ {
		fileData = append(fileData, pieces[i]...)
	}

	return fileData
}

func sendBlockRequest(conn net.Conn, pieceIndex, offset, blockSize int) error {
	request := make([]byte, PAYLOAD_BYTES)
	binary.BigEndian.PutUint32(request[LengthIndexStart:LengthIndexEnd], 13)
	request[4] = REQUEST_ID
	binary.BigEndian.PutUint32(request[PieceIndexStart:PieceIndexEnd], uint32(pieceIndex))
	binary.BigEndian.PutUint32(request[OffsetIndexStart:OffsetIndexEnd], uint32(offset))
	binary.BigEndian.PutUint32(request[BlocksizeIndexStart:BlocksizeIndexEnd], uint32(blockSize))

	log.Printf("Sending request for piece index: %d, block offset: %d, block size: %d",
		pieceIndex, offset, blockSize)

	_, err := conn.Write(request)
	return err
}
