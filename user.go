package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	keySize   = 32
	nonceSize = 12
)

var (
	ErrInvalidFormat = errors.New("invalid cookie format")
	ErrCrypto        = errors.New("crypto error")
)

type User struct {
	Name          string `json:"name"`
	CharacterName string `json:"character_name"`
	IsGameMaster  bool   `json:"is_game_master"`
	IPAddress     string `json:"ip_address"`
}

func (u *User) CookieValue(secret []byte) (string, error) {
	if len(secret) != keySize {
		return "", fmt.Errorf("crypto error: expecting %d byte secret key", keySize)
	}

	userBytes, err := json.Marshal(u)
	if err != nil {
		return "", fmt.Errorf("failed to marshal user JSON: %w", err)
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return "", fmt.Errorf("%w: failed to configure AES: %w", ErrCrypto, err)
	}

	nonce := make([]byte, nonceSize)

	n, err := rand.Read(nonce)
	if n != nonceSize {
		return "", fmt.Errorf("%w: wrong number of nonce bytes (%d)", ErrCrypto, nonceSize)
	} else if err != nil {
		return "", fmt.Errorf("%w: failed to generate nonce: %w", ErrCrypto, err)
	}

	nonceStr := hex.EncodeToString(nonce)

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: failed to Configure AES-GCM: %w", ErrCrypto, err)
	}

	userBytesCrypted := aesgcm.Seal(nil, nonce, userBytes, nil)

	userB64 := base64.StdEncoding.EncodeToString(userBytesCrypted)

	hash := sha256.New()
	hash.Write([]byte(userB64))
	sumStr := hex.EncodeToString(hash.Sum(nil))

	return userB64 + "||" + nonceStr + "||" + sumStr, nil
}

func (u *User) DataCookie(secret []byte) (*http.Cookie, error) {
	cookieVal, err := u.CookieValue(secret)
	if err != nil {
		return nil, err
	}

	cookie := &http.Cookie{
		Name:     CookieData,
		Value:    cookieVal,
		Expires:  time.Now().Add(24 * time.Hour),
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}

	return cookie, nil
}

func UserFromCookie(value string, secret []byte) (*User, error) {
	cookieParts := strings.Split(value, "||")
	if len(cookieParts) != 3 {
		return nil, fmt.Errorf("%w: expecting 3 parts", ErrInvalidFormat)
	}

	nonce, err := hex.DecodeString(cookieParts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: invalid nonce encoding: %w", ErrCrypto, err)
	}

	if len(nonce) != nonceSize {
		return nil, fmt.Errorf("%w: invalid nonce size", ErrInvalidFormat)
	}

	if len(cookieParts[2]) != 64 {
		return nil, fmt.Errorf("%w: invalid checksum size", ErrInvalidFormat)
	}

	hash := sha256.New()
	hash.Write([]byte(cookieParts[0]))
	expectedSum := hex.EncodeToString(hash.Sum(nil))

	if expectedSum != cookieParts[2] {
		return nil, fmt.Errorf("%w: checksum invalid", ErrInvalidFormat)
	}

	unwrapped, err := base64.StdEncoding.DecodeString(cookieParts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: data encoding corrupted", ErrInvalidFormat)
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to set up AES: %w", ErrCrypto, err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to Configure AES-GCM: %w", ErrCrypto, err)
	}

	plaintext, err := aesgcm.Open(nil, nonce, unwrapped, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decrypt cookie: %w", ErrCrypto, err)
	}

	data := &User{}
	if err := json.Unmarshal(plaintext, data); err != nil {
		return nil, fmt.Errorf("%w: data corrupted: %w", ErrInvalidFormat, err)
	}

	return data, nil
}

func UserFromContext(req *http.Request) *User {
	user, ok := req.Context().Value(userKey).(*User)
	if !ok {
		return nil
	}

	return user
}
