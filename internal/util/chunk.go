
package util

func Chunk(data []byte, max int) [][]byte {
	if len(data) <= max { return [][]byte{data} }
	parts := make([][]byte, 0, (len(data)+max-1)/max)
	for off := 0; off < len(data); off += max {
		end := off + max
		if end > len(data) { end = len(data) }
		parts = append(parts, data[off:end])
	}
	return parts
}
