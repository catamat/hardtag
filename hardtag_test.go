package hardtag

import (
	"strings"
	"testing"
)

// TestGenerateFromMAC checks that the GenerateFromMAC function returns a valid tag.
func TestGenerateFromMAC(t *testing.T) {
	tag, err := GenerateFromMAC()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if tag == "" {
		t.Fatal("Expected a non-empty tag, got an empty string")
	}

	// If the tag contains multiple MAC addresses check that they are separated by '|'
	if len(tag) > 17 && !strings.Contains(tag, "|") {
		t.Errorf("Expected tag to contain MAC addresses separated by '|', got %s", tag)
	}
}

// TestHashWithSHA256 checks that the HashWithSHA256 function returns the correct hash.
func TestHashWithSHA256(t *testing.T) {
	tag := "00:1A:2B:3C:4D:5E|00:1A:2B:3C:4D:6F"
	expectedHash := "3f0fe0bc0cca0c1a1838c21243ab6a010f8cb775f447f6b6f1ff0548e04420d6"

	hashedTag := HashWithSHA256(tag)

	if hashedTag != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, hashedTag)
	}
}

// TestGenerateFromMACAndHash integrates GenerateFromMAC and HashWithSHA256.
func TestGenerateFromMACAndHash(t *testing.T) {
	tag, err := GenerateFromMAC()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if tag == "" {
		t.Fatal("Expected a non-empty tag, got an empty string")
	}

	hashedTag := HashWithSHA256(tag)

	if hashedTag == "" {
		t.Error("Expected a non-empty hash, got an empty string")
	}
}
