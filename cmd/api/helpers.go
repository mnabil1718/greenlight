package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

func (app *application) getIdFromRequestContext(request *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(request.Context())
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64) // int64
	if err != nil || id < 1 {
		return 0, errors.New("invalid id")
	}

	return id, nil
}

type envelope map[string]interface{}

func (app *application) writeJSON(writer http.ResponseWriter, code int, data envelope, headers http.Header) error {
	resp, err := json.Marshal(data)
	if err != nil {
		return err
	}

	resp = append(resp, '\n')

	for key, value := range headers {
		writer.Header()[key] = value
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	writer.Write(resp)

	return nil

}

func (app *application) readJSON(writer http.ResponseWriter, request *http.Request, destination any) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1_048_576
	request.Body = http.MaxBytesReader(writer, request.Body, int64(maxBytes))

	dec := json.NewDecoder(request.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(destination) // receiver must be a pointer
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)

		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		default:
			return err
		}
	}

	// ensure theres nothing left in the decoder stream
	// if there is, meaning client send more than one JSON object, which is invalid
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}
