package utils

import (
	"github.com/google/uuid"
)

// GenerateClandCID generates a cland-cid in c+uuid format
func GenerateClandCID() string {
	return "c" + uuid.New().String()
}

// GenerateSessionID generates a session ID in se+uuid format
func GenerateSessionID() string {
	return "se" + uuid.New().String()
}

// GenerateSubSessionID generates a sub-session ID in ss+uuid format
func GenerateSubSessionID() string {
	return "ss" + uuid.New().String()
}

// IsValidClandCID validates a cland-cid format
func IsValidClandCID(id string) bool {
	if len(id) < 37 { // c + 36 chars for UUID
		return false
	}
	if id[0] != 'c' {
		return false
	}
	_, err := uuid.Parse(id[1:])
	return err == nil
}

// IsValidSessionID validates a session ID format
func IsValidSessionID(id string) bool {
	if len(id) < 38 { // se + 36 chars for UUID
		return false
	}
	if id[:2] != "se" {
		return false
	}
	_, err := uuid.Parse(id[2:])
	return err == nil
}

// IsValidSubSessionID validates a sub-session ID format
func IsValidSubSessionID(id string) bool {
	if len(id) < 38 { // ss + 36 chars for UUID
		return false
	}
	if id[:2] != "ss" {
		return false
	}
	_, err := uuid.Parse(id[2:])
	return err == nil
}
