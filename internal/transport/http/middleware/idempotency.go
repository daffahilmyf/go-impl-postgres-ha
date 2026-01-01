package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	nethttp "net/http"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/transport/http/response"
	"github.com/gin-gonic/gin"
)

const (
	IdempotencyKeyCtx  = "idempotency_key"
	IdempotencyHashCtx = "idempotency_hash"
)

func IdempotencyRequired(allowBypass bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		idempotencyKey := c.GetHeader("Idempotency-Key")
		if idempotencyKey == "" {
			idempotencyKey = c.GetHeader("X-Idempotency-Key")
		}
		bypass := c.GetHeader("X-Test-Bypass-Idempotency")
		if idempotencyKey == "" && (!allowBypass || bypass != "true") {
			response.RespondError(c, nethttp.StatusBadRequest, "idempotency key is required")
			c.Abort()
			return
		}

		if idempotencyKey != "" {
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				response.RespondError(c, nethttp.StatusBadRequest, "invalid request body")
				c.Abort()
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			sum := sha256.Sum256(body)
			c.Set(IdempotencyKeyCtx, idempotencyKey)
			c.Set(IdempotencyHashCtx, hex.EncodeToString(sum[:]))
		} else {
			c.Set(IdempotencyKeyCtx, "")
			c.Set(IdempotencyHashCtx, "")
		}

		c.Next()
	}
}
