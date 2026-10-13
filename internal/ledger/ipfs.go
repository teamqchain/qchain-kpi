package ledger

import "fmt"

// IPFS storage of one file added the way QChain's backend adds an envelope
// (go-ipfs-api Shell.Add with Kubo's defaults: CIDv0, dag-pb nodes with UnixFS
// data, no raw leaves, 256 KiB fixed-size chunks, balanced layout). Checked
// against the live store in vmcapture/ (single blocks of several sizes and a
// six-chunk file).

// ChunkSize is Kubo's default chunker (size-262144).
const ChunkSize = 256 * 1024

// maxLinks is the balanced layout's links per node; one level suffices for
// files up to 174 chunks (about 43.5 MiB).
const maxLinks = 174

// leafBlock is the block of a file (or chunk) of l bytes stored as one dag-pb
// node: PBNode{Data: UnixFS{Type: File, Data: bytes, filesize}}.
func leafBlock(l int) int {
	inner := vf(1, 2) + ld(2, l) + vf(3, uint64(l)) // Type File (2), Data, filesize
	return ld(1, inner)
}

// IPFSBlocks returns the sizes of the blocks IPFS stores for a file of l
// bytes: one block up to ChunkSize, otherwise one block per chunk plus a root
// node linking them (each link: 34-byte CIDv0 hash, empty name, cumulative
// size; UnixFS data: Type, filesize and every chunk's size).
func IPFSBlocks(l int) ([]int, error) {
	if l <= ChunkSize {
		return []int{leafBlock(l)}, nil
	}
	var chunks []int
	for rest := l; rest > 0; rest -= ChunkSize {
		chunks = append(chunks, min(rest, ChunkSize))
	}
	if len(chunks) > maxLinks {
		return nil, fmt.Errorf("ipfs: %d bytes need a deeper tree than one level (%d chunks)", l, len(chunks))
	}
	root := 0
	data := vf(1, 2) + vf(3, uint64(l))
	blocks := make([]int, 0, len(chunks)+1)
	for _, c := range chunks {
		leaf := leafBlock(c)
		blocks = append(blocks, leaf)
		root += ld(2, ld(1, 34)+ld(2, 0)+vf(3, uint64(leaf)))
		data += vf(4, uint64(c))
	}
	root += ld(1, data)
	return append([]int{root}, blocks...), nil
}

// OnDisk is the space the blocks take in Kubo's flatfs datastore on a file
// system that allocates whole blocks of fsBlock bytes (ext4: 4,096): one
// file per IPFS block.
func OnDisk(blocks []int, fsBlock int) int {
	s := 0
	for _, b := range blocks {
		s += (b + fsBlock - 1) / fsBlock * fsBlock
	}
	return s
}
