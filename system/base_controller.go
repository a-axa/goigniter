package system

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type BaseController struct{}

func (bc *BaseController) Respond(w http.ResponseWriter, status int, message string, data interface{}) error {
	return Success(w, status, message, data)
}

func (bc *BaseController) RespondError(w http.ResponseWriter, status int, message string) error {
	return Error(w, status, message)
}

func (bc *BaseController) RespondValidation(w http.ResponseWriter, errors interface{}) error {
	return ValidationError(w, errors)
}

func (bc *BaseController) BodyParse(w http.ResponseWriter, r *http.Request, out interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		_ = Error(w, http.StatusBadRequest, "Invalid request body")
		return err
	}

	if err := validator.New().Struct(out); err != nil {
		_ = ValidationError(w, ParseValidationErrors(err))
		return err
	}

	return nil
}

func (bc *BaseController) CurrentUser(r *http.Request) *JWTClaims {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		return nil
	}
	return claims
}
