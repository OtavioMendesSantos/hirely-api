package dto

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Status  string   `json:"status"`
	Details []string `json:"details,omitempty"`
}

func WriteError(c *gin.Context, code int, message string, status string) {
	c.JSON(code, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Status:  status,
		},
	})
}

// HandleValidationError checks if err is validator.ValidationErrors and returns a standard error
func HandleValidationError(c *gin.Context, err error) {
	var valErrors validator.ValidationErrors
	if errors.As(err, &valErrors) {
		var details []string
		for _, v := range valErrors {
			details = append(details, fmt.Sprintf("Field '%s' failed on '%s' tag", v.Field(), v.Tag()))
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    http.StatusBadRequest,
				Message: "Validation failed",
				Status:  "INVALID_ARGUMENT",
				Details: details,
			},
		})
		return
	}

	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error: ErrorDetail{
			Code:    http.StatusBadRequest,
			Message: "Invalid JSON payload or format",
			Status:  "INVALID_ARGUMENT",
			Details: []string{err.Error()},
		},
	})
}
