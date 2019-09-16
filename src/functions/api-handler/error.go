package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maddiesch/automatic-reminders/auto"
	"github.com/maddiesch/serverless"
)

const (
	errCodeInternalServerError = "internal_server_error"
	errCodeBadRequest          = "bad_request"
	errCodeNotFound            = "not_found"
)

func respondWithError(c *gin.Context, err error) {
	if err, ok := err.(*Error); ok {
		validationErr := serverless.GetValidator().Struct(err)
		if validationErr != nil {
			panic(err)
		}
		c.AbortWithStatusJSON(err.Status, err)
	} else if err == auto.ErrRecordNotFound {
		respondWithError(c, &Error{
			Status: http.StatusNotFound,
			Title:  "Record not found",
			Detail: "The requested resource could not be found",
			Code:   errCodeNotFound,
		})
	} else {
		respondWithError(c, &Error{
			Status: http.StatusInternalServerError,
			Title:  "Internal Server Error",
			Detail: "An unknown error occurred.",
			Code:   errCodeInternalServerError,
			Meta: map[string]interface{}{
				"SubError": fmt.Sprintf("%v", err),
			},
			SubError: err,
		})
	}
}

// Error is an API response error code
type Error struct {
	Status   int                    `validate:"required,min=200,max=599"`
	Title    string                 `json:",omitempty" validate:"max=128"`
	Detail   string                 `json:",omitempty"`
	Code     string                 `json:",omitempty"`
	Meta     map[string]interface{} `json:",omitempty"`
	SubError error                  `json:"-"`
}

func (e *Error) Error() string {
	data, _ := json.Marshal(e)
	return string(data)
}

func reportError(err error, handled bool) {
	serverless.GetLogger().Printf("[ERROR] - %v", err)
}
