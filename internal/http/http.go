package http

import (
	"github.com/gin-gonic/gin"
	"github.com/kerem-kaynak/katalog/internal/context"
	"github.com/kerem-kaynak/katalog/internal/http/middleware"
)

type APIService struct {
	engine  *gin.Engine
	context *context.Context
}

func NewHTTPService(ctx *context.Context) *APIService {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORSMiddleware())

	service := &APIService{
		engine:  engine,
		context: ctx,
	}
	service.setupRoutes()
	return service
}

func (h *APIService) Engine() *gin.Engine {
	return h.engine
}

func (h *APIService) setupRoutes() {
	h.setupAuthRoutes()
	h.setupProjectRoutes()
	h.setupDatasetRoutes()
	h.setupTableRoutes()
	h.setupColumnRoutes()
}

func (h *APIService) setupAuthRoutes() {}

func (h *APIService) setupProjectRoutes() {}

func (h *APIService) setupDatasetRoutes() {}

func (h *APIService) setupTableRoutes() {}

func (h *APIService) setupColumnRoutes() {}
