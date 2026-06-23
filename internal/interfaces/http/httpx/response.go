package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type errorBody struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type pageBody struct {
	Items  any   `json:"items"`
	Total  int64 `json:"total"`
	Number int   `json:"number"`
	Size   int   `json:"size"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

func Page(c *gin.Context, items any, total int64, number, size int) {
	c.JSON(http.StatusOK, pageBody{Items: items, Total: total, Number: number, Size: size})
}

func Error(c *gin.Context, err error) {
	status := statusFor(errs.KindOf(err))
	code := string(errs.KindOf(err))
	message := http.StatusText(status)

	if e, ok := errs.As(err); ok {
		if e.Code != "" {
			code = e.Code
		}
		if e.Message != "" {
			message = e.Message
		}
	}

	c.AbortWithStatusJSON(status, errorBody{Error: errorPayload{Code: code, Message: message}})
}

func statusFor(kind errs.Kind) int {
	switch kind {
	case errs.KindValidation:
		return http.StatusBadRequest
	case errs.KindNotFound:
		return http.StatusNotFound
	case errs.KindConflict:
		return http.StatusConflict
	case errs.KindUnauthorized:
		return http.StatusUnauthorized
	case errs.KindForbidden:
		return http.StatusForbidden
	case errs.KindOutOfStock:
		return http.StatusConflict
	case errs.KindPayment:
		return http.StatusPaymentRequired
	case errs.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
