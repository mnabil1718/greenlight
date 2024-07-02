package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/mnabil1718/greenlight/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Activated bool      `json:"activated"`
	Password  password  `json:"-"`
	Version   int32     `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

var AnonymousUser = &User{}

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

type password struct {
	plaintext *string // pointer to differentiate "" and nil in json
	hash      []byte
}

func (password *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	password.plaintext = &plaintextPassword
	password.hash = hash

	return nil
}

func (password *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(password.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")
	ValidateEmail(v, user.Email)
	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}
	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}

type UserModel struct {
	DB *sql.DB
}

func (model UserModel) Insert(user *User) error {
	SQL := `INSERT INTO users (name, email, password)
			VALUES ($1, $2, $3)
			RETURNING id, created_at, version`

	args := []interface{}{user.Name, user.Email, user.Password.hash}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := model.DB.QueryRowContext(ctx, SQL, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), `violates unique constraint "users_email_key"`):
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (model UserModel) GetByEmail(email string) (*User, error) {
	SQL := `SELECT id, name, email, password, activated, created_at, version 
			FROM users WHERE email = $1`

	user := &User{}
	args := []interface{}{email}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := model.DB.QueryRowContext(ctx, SQL, args...).Scan(&user.ID, &user.Name, &user.Email, &user.Password.hash, &user.Activated, &user.CreatedAt, &user.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return user, nil
}

func (model UserModel) Update(user *User) error {
	// even if we passed email param, if the value
	// is the same as the previous value in the record,
	// then UNIQUE constraint won't kick in

	SQL := `UPDATE users 
			SET name=$1, email=$2, password=$3, activated=$4, version=version+1
			WHERE id=$5 AND version=$6
			RETURNING version`

	args := []interface{}{user.Name, user.Email, user.Password.hash, user.Activated, user.ID, user.Version}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := model.DB.QueryRowContext(ctx, SQL, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (model UserModel) GetForToken(scope string, tokenPlainText string) (*User, error) {
	SQL := `SELECT u.id, u.name, u.email, u.password, u.activated, u.version, u.created_at FROM users u 
			INNER JOIN tokens t ON t.user_id=u.id
			WHERE t.hash=$1 AND t.scope=$2 AND expiry_time > NOW()`

	hashArray := sha256.Sum256([]byte(tokenPlainText))

	user := &User{}

	args := []interface{}{hashArray[:], scope}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := model.DB.QueryRowContext(ctx, SQL, args...).Scan(&user.ID, &user.Name, &user.Email, &user.Password.hash, &user.Activated, &user.Version, &user.CreatedAt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return user, nil
}

type MockUserModel struct{}

func (model MockUserModel) Insert(user *User) error {
	return nil
}

func (model MockUserModel) GetByEmail(email string) (*User, error) {
	user := &User{
		ID:        1,
		Name:      "Elole Kusk",
		Email:     "elole@gmail.com",
		Activated: true,
		Version:   1,
		CreatedAt: time.Now(),
	}

	return user, nil
}

func (model MockUserModel) Update(user *User) error {
	return nil
}

func (model MockUserModel) GetForToken(scope string, tokenPlainText string) (*User, error) {
	user := &User{
		ID:        1,
		Name:      "Elole Kusk",
		Email:     "elole@gmail.com",
		Activated: true,
		Version:   1,
		CreatedAt: time.Now(),
	}
	return user, nil
}
