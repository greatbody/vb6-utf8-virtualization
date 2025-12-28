package transcoder

import (
	"bytes"
	"io"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// NormalizeToUTF8 converts GB18030 data to UTF-8.
func NormalizeToUTF8(data []byte) ([]byte, error) {
	enc := DetectEncoding(data)
	if enc == EncodingUTF8 {
		// Already UTF-8, remove BOM if present
		return bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}), nil
	}

	// Convert GB18030 to UTF-8
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewDecoder())
	return io.ReadAll(reader)
}

// ConvertToGB18030 converts UTF-8 data back to GB18030.
func ConvertToGB18030(data []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewEncoder())
	return io.ReadAll(reader)
}

// StreamNormalizeToUTF8 provides a reader that converts on the fly.
// NOTE: This might be tricky for random access, but useful for sequential reads.
func StreamNormalizeToUTF8(r io.Reader) io.Reader {
	// For simplicity in this proxy, we might stick to buffer-based for Seek support
	// but the requirement mentioned stream processing for boundary handling.
	return transform.NewReader(r, simplifiedchinese.GB18030.NewDecoder())
}
