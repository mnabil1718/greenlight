package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"time"

	"github.com/mnabil1718/greenlight/internal/validator"
)

const (
	ScopeActivation = "activation"
)

type Token struct {
	Plaintext  string
	Hash       []byte
	UserID     int64
	ExpiryTime time.Time
	Scope      string
}

func generateToken(userID int64, scope string, ttl time.Duration) (*Token, error) {
	token := &Token{
		UserID:     userID,
		Scope:      scope,
		ExpiryTime: time.Now().Add(ttl),
	}

	randomBytes := make([]byte, 16)  // allocate []byte with length of 16
	_, err := rand.Read(randomBytes) // fill it with random bytes entropy
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	// array to slice conversion cannot happen in one line
	// because to initialize a slice buffer, memory allocation
	// has to exists on the heap not the stack
	// see: https://stackoverflow.com/questions/28886616/convert-array-to-slice-in-go
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

type TokenModel struct {
	DB *sql.DB
}

func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 16 bytes long")
}

func (model TokenModel) Insert(token *Token) error {
	SQL := `INSERT INTO tokens (hash, user_id, expiry_time, scope)
			VALUES ($1, $2, $3, $4)`

	args := []interface{}{token.Hash, token.UserID, token.ExpiryTime, token.Scope}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := model.DB.ExecContext(ctx, SQL, args...)
	return err
}

func (model TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, scope, ttl)
	if err != nil {
		return nil, err
	}

	err = model.Insert(token)
	return token, err
}

func (model TokenModel) DeleteForAllUser(scope string, userID int64) error {
	SQL := `DELETE FROM tokens
			WHERE scope=$1 and user_id=$2`

	args := []interface{}{scope, userID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := model.DB.ExecContext(ctx, SQL, args...)
	return err
}
