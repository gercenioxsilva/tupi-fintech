package infrastructure

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type TLVDecoder struct{}

func (TLVDecoder) Decode(input string) (map[string]string, error) {
	cleaned := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(input), " ", ""))
	if cleaned == "" {
		return nil, fmt.Errorf("empty TLV payload")
	}
	if _, err := hex.DecodeString(cleaned); err != nil {
		return nil, fmt.Errorf("payload is not valid hexadecimal: %w", err)
	}

	result := make(map[string]string)
	for i := 0; i < len(cleaned); {
		tag, offset, err := readTag(cleaned, i)
		if err != nil {
			return nil, err
		}
		length, next, err := readLength(cleaned, offset)
		if err != nil {
			return nil, err
		}
		end := next + length*2
		if end > len(cleaned) {
			return nil, fmt.Errorf("value for tag %s exceeds payload size", tag)
		}
		result[tag] = cleaned[next:end]
		i = end
	}
	return result, nil
}

func readTag(payload string, start int) (string, int, error) {
	if start+2 > len(payload) {
		return "", 0, fmt.Errorf("unexpected end of payload while reading tag")
	}
	firstByte := payload[start : start+2]
	tag := firstByte
	offset := start + 2
	decoded, err := hex.DecodeString(firstByte)
	if err != nil {
		return "", 0, fmt.Errorf("invalid tag prefix: %w", err)
	}
	if decoded[0]&0x1F == 0x1F {
		if offset+2 > len(payload) {
			return "", 0, fmt.Errorf("unexpected end of payload while reading multi-byte tag")
		}
		tag += payload[offset : offset+2]
		offset += 2
	}
	return tag, offset, nil
}

func readLength(payload string, start int) (int, int, error) {
	if start+2 > len(payload) {
		return 0, 0, fmt.Errorf("unexpected end of payload while reading length")
	}
	var length int
	_, err := fmt.Sscanf(payload[start:start+2], "%02X", &length)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid TLV length: %w", err)
	}
	return length, start + 2, nil
}
