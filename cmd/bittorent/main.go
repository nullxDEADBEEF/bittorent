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
	downloadCmd := flag.NewFlagSet("download_piece", flag.ExitOnError)
	outputFile := downloadCmd.String("o", "", "output file path")

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
		downloadCmd.Parse(os.Args[2:])

		if *outputFile == "" {
			fmt.Println("Output file path is required")
			downloadCmd.PrintDefaults()
			os.Exit(1)
		}

		// create directory if it doesnt exist
		outputDir := filepath.Dir(*outputFile)
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			err := os.MkdirAll(outputDir, os.ModePerm)
			if err != nil {
				fmt.Println("Failed to create directory", err)
				return
			}
		}

		// create file if it doesnt exist
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Println("Failed to create output file", err)
			return
		}
		defer file.Close()

		pieceIndex, err := strconv.Atoi(os.Args[5])
		if err != nil {
			fmt.Println(err)
			return
		}

		torrentPath := os.Args[4]
		pieceData := downloadPiece(torrentPath, pieceIndex)
		file.Write(pieceData)

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
