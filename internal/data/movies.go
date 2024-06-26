package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mnabil1718/greenlight/internal/validator"
)

type Movie struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
	CreatedAt time.Time `json:"-"`
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")
	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")
	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")
	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}

type MovieModel struct {
	DB *sql.DB
}

func (model MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {

	// id is included in ORDER BY to ensure sorting produces the exact order, see: https://www.postgresql.org/docs/current/queries-order.html#QUERIES-ORDER
	// don't worry, string interpolation is already sanitized
	SQL := fmt.Sprintf(`
			SELECT COUNT(*) OVER(), id, title, year, runtime, genres, version, created_at
			FROM movies
			WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
			AND (genres @> $2 OR $2 = '{}')
			ORDER BY %s %s, id ASC
			LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	args := []interface{}{title, genres, filters.limit(), filters.offset()}

	// the timeout starts right after creating this context
	//  any other operation after this will be counted on timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := model.DB.QueryContext(ctx, SQL, args...)
	if err != nil {
		return nil, Metadata{}, err // Metadata cannot be nil because its not a pointer
	}
	defer rows.Close()

	totalRecords := 0
	movies := []*Movie{}

	for rows.Next() {
		movie := &Movie{}

		m := pgtype.NewMap()
		var genres []string

		err := rows.Scan(&totalRecords, &movie.ID, &movie.Title, &movie.Year, &movie.Runtime, m.SQLScanner(&genres), &movie.Version, &movie.CreatedAt)
		// error from a single row
		// e.g. error from the scanner
		if err != nil {
			return nil, Metadata{}, err
		}
		movie.Genres = genres
		movies = append(movies, movie)
	}

	// collecting errors during the iterations
	// e.g. connection issues, unexpected errors
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return movies, metadata, nil
}

func (model MovieModel) Insert(movie *Movie) error {
	SQL := `INSERT INTO movies (title, year, runtime, genres) 
			VALUES ($1, $2, $3, $4) 
			RETURNING id, created_at, version`

	args := []interface{}{movie.Title, movie.Year, movie.Runtime, movie.Genres}
	// the timeout starts right after creating this context
	//  any other operation after this will be counted on timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := model.DB.QueryRowContext(ctx, SQL, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
	if err != nil {
		return err
	}
	return nil
}

func (model MovieModel) Get(id int64) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	movie := &Movie{}
	SQL := `SELECT id,title,year,runtime, genres,version,created_at
			FROM movies
			WHERE id=$1`

	args := []interface{}{id}

	// cannot scan directly into []string, see: https://github.com/jackc/pgx/issues/1779
	m := pgtype.NewMap()
	var genres []string

	// the timeout starts right after creating this context
	//  any other operation after this will be counted on timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := model.DB.QueryRowContext(ctx, SQL, args...).Scan(&movie.ID, &movie.Title, &movie.Year, &movie.Runtime, m.SQLScanner(&genres), &movie.Version, &movie.CreatedAt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound

		default:
			return nil, err
		}
	}

	movie.Genres = genres
	return movie, nil
}

func (model MovieModel) Update(movie *Movie) error {
	SQL := `UPDATE movies
			SET title=$1, year=$2, runtime=$3, genres=$4, version = version + 1
			WHERE id=$5 AND version = $6
			RETURNING version`

	args := []interface{}{movie.Title, movie.Year, movie.Runtime, movie.Genres, movie.ID, movie.Version}
	// the timeout starts right after creating this context
	//  any other operation after this will be counted on timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := model.DB.QueryRowContext(ctx, SQL, args...).Scan(&movie.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (model MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	SQL := `DELETE FROM movies WHERE id=$1`
	// the timeout starts right after creating this context
	//  any other operation after this will be counted on timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := model.DB.ExecContext(ctx, SQL, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

type MockMovieModel struct{}

func (m MockMovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {
	return nil, Metadata{}, nil
}

func (m MockMovieModel) Insert(movie *Movie) error {
	return nil
}

func (m MockMovieModel) Get(id int64) (*Movie, error) {
	return nil, nil
}

func (m MockMovieModel) Update(movie *Movie) error {
	return nil
}

func (m MockMovieModel) Delete(id int64) error {
	return nil
}
