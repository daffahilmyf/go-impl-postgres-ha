package handlers

import "github.com/gin-gonic/gin"

type Router struct {
	handler *Handler
}

func NewRouter(handler *Handler) *Router {
	return &Router{handler: handler}
}

func (r *Router) RegisterRoutes(engine *gin.Engine, idempotency gin.HandlerFunc) {
	engine.GET("/healthz", r.handler.health)

	api := engine.Group("/api")
	users := api.Group("/users")
	users.POST("", idempotency, r.handler.createUser)
	users.GET("", r.handler.listUsers)
	users.GET("/:id", r.handler.getUser)
	users.PATCH("/:id", r.handler.updateUser)
	users.DELETE("/:id", r.handler.deleteUser)
}
