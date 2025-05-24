package auth_test

import (
	"testing"

	"github.com/veryfrank/Chirpy/internal/auth"
)

func TestHashing(t *testing.T) {
	pw := "AbsoluutEenWachtwoord"
	hash, err := auth.HashPassword(pw)

	if err != nil {
		t.Error(err)
	}

	err = auth.CheckPasswordHash(hash, pw)
	if err != nil {
		t.Error(err)
	}
}
