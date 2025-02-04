package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func main() {
	command := os.Args[1]

	// flags for commands
	downloadPieceCmd := flag.NewFlagSet("download_piece", flag.ExitOnError)
	downloadCmd := flag.NewFlagSet("download", flag.ExitOnError)

	switch command {
	case "decode":
		fmt.Println(handleDecode(os.Args[2]))
	case "info":
		handleInfo(os.Args[2])
	case "peers":
		peers := handlePeers(os.Args[2])
		for _, peer := range peers {
			fmt.Println(peer)
		}
	case "handshake":
		conn, peerID := handleHandshake(os.Args[2], os.Args[3])
		fmt.Println("Peer ID: " + peerID)

		defer conn.Close()
	case "download_piece":
		outputFile := downloadPieceCmd.String("o", "", "output file path")
		downloadPieceCmd.Parse(os.Args[2:])

		if *outputFile == "" {
			fmt.Println("Output file path is required")
			downloadPieceCmd.PrintDefaults()
			os.Exit(1)
		}

		file := createFile(*outputFile)
		defer file.Close()

		pieceIndex, err := strconv.Atoi(os.Args[5])
		if err != nil {
			fmt.Println(err)
			return
		}

		torrentPath := os.Args[4]
		pieceData := downloadPiece(torrentPath, pieceIndex)
		file.Write(pieceData)
	case "download":
		outputFile := downloadCmd.String("o", "", "output file path")
		downloadCmd.Parse(os.Args[2:])

		if *outputFile == "" {
			fmt.Println("Output file path is required")
			downloadPieceCmd.PrintDefaults()
			os.Exit(1)
		}

		file := createFile(*outputFile)
		defer file.Close()

		torrentPath := os.Args[4]
		fileData := download(torrentPath)
		file.Write(fileData)
	case "magnet_parse":
		break
		//magnetLink := os.Args[2]
		//handleMagnetLink(magnetLink)
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}

func createFile(outputFile string) *os.File {

	// create directory if it doesnt exist
	outputDir := filepath.Dir(outputFile)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			fmt.Println("Failed to create directory", err)
			return nil
		}
	}

	// create file if it doesnt exist
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Failed to create output file", err)
		return nil
	}

	return file
}
