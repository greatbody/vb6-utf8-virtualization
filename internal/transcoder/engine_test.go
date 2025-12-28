package transcoder

import (
	"bytes"
	"testing"
)

func TestNormalizeToUTF8(t *testing.T) {
	// GB18030 "你好"
	gbData := []byte{0xC4, 0xE3, 0xBA, 0xC3}
	utf8Data := []byte("你好")

	// Test GB to UTF-8
	got, err := NormalizeToUTF8(gbData)
	if err != nil {
		t.Fatalf("NormalizeToUTF8(gbData) failed: %v", err)
	}
	if !bytes.Equal(got, utf8Data) {
		t.Errorf("NormalizeToUTF8(gbData) = %v, want %v", got, utf8Data)
	}

	// Test UTF-8 with BOM
	bomUtf8 := append([]byte{0xEF, 0xBB, 0xBF}, utf8Data...)
	got, err = NormalizeToUTF8(bomUtf8)
	if err != nil {
		t.Fatalf("NormalizeToUTF8(bomUtf8) failed: %v", err)
	}
	if !bytes.Equal(got, utf8Data) {
		t.Errorf("NormalizeToUTF8(bomUtf8) should strip BOM, got %v", got)
	}
}

func TestConvertToGB18030(t *testing.T) {
	utf8Data := []byte("你好")
	gbData := []byte{0xC4, 0xE3, 0xBA, 0xC3}

	got, err := ConvertToGB18030(utf8Data)
	if err != nil {
		t.Fatalf("ConvertToGB18030(utf8Data) failed: %v", err)
	}
	if !bytes.Equal(got, gbData) {
		t.Errorf("ConvertToGB18030(utf8Data) = %v, want %v", got, gbData)
	}
}
