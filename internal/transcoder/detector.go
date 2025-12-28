package transcoder

import (
	"bytes"
	"unicode/utf8"
)

// Encoding represents the detected encoding of a file.
type Encoding int

const (
	EncodingUnknown Encoding = iota
	EncodingUTF8
	EncodingGB18030 // We'll use GB18030 as it's backward compatible with GB2312/GBK
)

const (
	// DetectionBufferSize is the amount of data we'll read to detect encoding.
	DetectionBufferSize = 4096
)

// DetectEncoding detects if the given data is UTF-8 or GB18030.
func DetectEncoding(data []byte) Encoding {
	// 1. Check for BOM (EF BB BF)
	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		return EncodingUTF8
	}

	// 2. Check if valid UTF-8
	if utf8.Valid(data) {
		return EncodingUTF8
	}

	// 3. Fallback to GB18030 (assuming Chinese environment as per requirements)
	return EncodingGB18030
}
