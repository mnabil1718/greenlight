package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type MovieModelInterface interface {
	GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error)
	Insert(movie *Movie) error
	Get(id int64) (*Movie, error)
	Update(movie *Movie) error
	Delete(id int64) error
}

type UsersModelInterface interface {
	Insert(user *User) error
	GetByEmail(email string) (*User, error)
	Update(user *User) error
	GetForToken(scope string, tokenPlainText string) (*User, error)
}

type Models struct {
	Movies MovieModelInterface
	Users  UsersModelInterface
	Tokens TokenModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db},
		Users:  UserModel{DB: db},
		Tokens: TokenModel{DB: db},
	}
}

func NewMockModels() Models {
	return Models{
		Movies: MockMovieModel{},
		Users:  MockUserModel{},
	}
}
