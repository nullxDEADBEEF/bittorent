## How to run

```
go run . decode <bencoded string>
go run . info <path to torrent file>
go run . peers <path to torrent file>
go run . handshake <path to torrent file> <peer_ip>:<peer_port>
go run . download_piece -o <output path> <path to torrent file> <piece_index>
go run . download -o <output path> <path to torrent>
```