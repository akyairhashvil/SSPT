package util

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    uint32 = 1
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
)

func HashPassphrase(pass string) string {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return ""
	}
	hash := argon2.IDKey([]byte(pass), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
}

func VerifyPassphrase(hash, pass string) bool {
	ok, _, _ := verifyPassphrase(hash, pass)
	return ok
}

func VerifyPassphraseWithUpgrade(hash, pass string) (bool, string) {
	ok, upgraded, newHash := verifyPassphrase(hash, pass)
	if !ok || !upgraded {
		return ok, ""
	}
	return true, newHash
}

func ValidatePassphrase(pass string) error {
	if len(pass) < 8 {
		return fmt.Errorf("passphrase must be at least 8 characters")
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range pass {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("passphrase must contain uppercase, lowercase, and digit")
	}
	return nil
}

func verifyPassphrase(encodedHash, pass string) (bool, bool, string) {
	if encodedHash == "" || pass == "" {
		return false, false, ""
	}
	if strings.HasPrefix(encodedHash, "$argon2id$") {
		ok, _ := verifyArgon2id(encodedHash, pass)
		return ok, false, ""
	}

	legacyHash, err := hex.DecodeString(encodedHash)
	if err != nil || len(legacyHash) != sha256.Size {
		return false, false, ""
	}
	sum := sha256.Sum256([]byte(pass))
	if subtle.ConstantTimeCompare(sum[:], legacyHash) != 1 {
		return false, false, ""
	}
	newHash := HashPassphrase(pass)
	if newHash == "" {
		return true, false, ""
	}
	return true, true, newHash
}

func verifyArgon2id(encodedHash, pass string) (bool, error) {
	params, salt, hash, err := parseArgon2idHash(encodedHash)
	if err != nil {
		return false, err
	}
	calculated := argon2.IDKey([]byte(pass), salt, params.time, params.memory, params.threads, uint32(len(hash)))
	if subtle.ConstantTimeCompare(calculated, hash) != 1 {
		return false, nil
	}
	return true, nil
}

type argon2Params struct {
	memory  uint32
	time    uint32
	threads uint8
}

func parseArgon2idHash(encodedHash string) (argon2Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argon2Params{}, nil, nil, errors.New("invalid argon2id hash format")
	}
	if parts[2] != "v=19" {
		return argon2Params{}, nil, nil, errors.New("unsupported argon2id version")
	}
	params, err := parseArgon2Params(parts[3])
	if err != nil {
		return argon2Params{}, nil, nil, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2Params{}, nil, nil, errors.New("invalid argon2id salt")
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2Params{}, nil, nil, errors.New("invalid argon2id hash")
	}
	if len(salt) == 0 || len(hash) == 0 {
		return argon2Params{}, nil, nil, errors.New("invalid argon2id payload")
	}
	return params, salt, hash, nil
}

func parseArgon2Params(params string) (argon2Params, error) {
	parts := strings.Split(params, ",")
	if len(parts) != 3 {
		return argon2Params{}, errors.New("invalid argon2id params")
	}
	values := make(map[string]string, 3)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return argon2Params{}, errors.New("invalid argon2id params")
		}
		values[kv[0]] = kv[1]
	}
	memory, err := strconv.ParseUint(values["m"], 10, 32)
	if err != nil {
		return argon2Params{}, errors.New("invalid argon2id memory")
	}
	time, err := strconv.ParseUint(values["t"], 10, 32)
	if err != nil {
		return argon2Params{}, errors.New("invalid argon2id time")
	}
	threads, err := strconv.ParseUint(values["p"], 10, 8)
	if err != nil {
		return argon2Params{}, errors.New("invalid argon2id threads")
	}
	return argon2Params{
		memory:  uint32(memory),
		time:    uint32(time),
		threads: uint8(threads),
	}, nil
}
