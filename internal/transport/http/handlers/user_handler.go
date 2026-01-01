package handlers

import (
	nethttp "net/http"
	"strconv"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/repository"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/domain/service"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/transport/http/middleware"
	"github.com/daffahilmyf/go-impl-postgres-ha/internal/transport/http/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	user  service.UserService
	store repository.Store
}

func NewHandler(user service.UserService, store repository.Store) *Handler {
	return &Handler{
		user:  user,
		store: store,
	}
}

type createUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

func (h *Handler) createUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondError(c, nethttp.StatusBadRequest, err.Error())
		return
	}

	idempotencyKey := c.GetString(middleware.IdempotencyKeyCtx)
	requestHash := c.GetString(middleware.IdempotencyHashCtx)

	user, alreadyExist, err := h.user.Create(c.Request.Context(), req.Name, req.Email, idempotencyKey, requestHash)
	if err != nil {
		if err == repository.ErrIdempotencyKeyConflict {
			response.RespondError(c, nethttp.StatusConflict, "idempotency key conflicts with request")
			return
		}
		response.RespondError(c, nethttp.StatusInternalServerError, "create failed")
		return
	}
	if alreadyExist {
		response.RespondOK(c, nethttp.StatusOK, user, nil)
		return
	}
	response.RespondOK(c, nethttp.StatusCreated, user, nil)
}

func (h *Handler) getUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RespondError(c, nethttp.StatusBadRequest, "invalid id")
		return
	}

	user, err := h.user.GetByID(c.Request.Context(), id)
	if err != nil {
		response.RespondError(c, nethttp.StatusNotFound, "not found")
		return
	}
	response.RespondOK(c, nethttp.StatusOK, user, nil)
}

type updateUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

func (h *Handler) updateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RespondError(c, nethttp.StatusBadRequest, "invalid id")
		return
	}
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondError(c, nethttp.StatusBadRequest, err.Error())
		return
	}

	user, err := h.user.Update(c.Request.Context(), id, req.Name, req.Email)
	if err != nil {
		response.RespondError(c, nethttp.StatusInternalServerError, "update failed")
		return
	}
	response.RespondOK(c, nethttp.StatusOK, user, nil)
}

func (h *Handler) deleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RespondError(c, nethttp.StatusBadRequest, "invalid id")
		return
	}

	if err := h.user.DeleteByID(c.Request.Context(), id); err != nil {
		response.RespondError(c, nethttp.StatusInternalServerError, "delete failed")
		return
	}
	response.RespondOK(c, nethttp.StatusOK, gin.H{"status": "deleted"}, nil)
}

func (h *Handler) listUsers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	cursor := c.Query("cursor")

	users, nextCursor, err := h.user.List(c.Request.Context(), limit, cursor)
	if err != nil {
		if err == repository.ErrInvalidCursor {
			response.RespondError(c, nethttp.StatusBadRequest, "invalid cursor")
			return
		}
		response.RespondError(c, nethttp.StatusInternalServerError, "list failed")
		return
	}
	meta := &response.Meta{NextCursor: nextCursor}
	response.RespondOK(c, nethttp.StatusOK, users, meta)
}

func (h *Handler) health(c *gin.Context) {
	if err := h.store.Ping(c.Request.Context()); err != nil {
		response.RespondOK(c, nethttp.StatusServiceUnavailable, gin.H{"status": "down"}, nil)
		return
	}
	response.RespondOK(c, nethttp.StatusOK, gin.H{"status": "ok"}, nil)
}
