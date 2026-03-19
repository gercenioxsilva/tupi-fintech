package infrastructure

import "testing"

func TestTLVDecoderDecode(t *testing.T) {
	payload := "5A0841111111111111115F24033012319F34031E0300"
	decoded, err := (TLVDecoder{}).Decode(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded["5A"] != "4111111111111111" {
		t.Fatalf("unexpected pan: %s", decoded["5A"])
	}
	if decoded["5F24"] != "301231" {
		t.Fatalf("unexpected expiry: %s", decoded["5F24"])
	}
	if decoded["9F34"] != "1E0300" {
		t.Fatalf("unexpected cvm: %s", decoded["9F34"])
	}
}
