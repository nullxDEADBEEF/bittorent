package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	command := os.Args[1]

	switch command {
	case "decode":
		bencodedValue := os.Args[2]

		index := 0
		decoded, err := decodeBencode(bencodedValue, &index)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	case "info":

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
