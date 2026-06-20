package migration

import "testing"

func TestChecksum(t *testing.T) {
	got := Checksum([]byte("CREATE TABLE users (id BIGSERIAL PRIMARY KEY);\n"))
	const want = "c828623fa456953143de66b66a7b435374be74e04075ab1b45496f44df7205b8"
	if got != want {
		t.Fatalf("Checksum() = %q, want %q", got, want)
	}
}

func TestChecksumEmptyContent(t *testing.T) {
	got := Checksum(nil)
	const want = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Fatalf("Checksum(nil) = %q, want %q", got, want)
	}
}
