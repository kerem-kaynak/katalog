package http

import (
	"github.com/gin-gonic/gin"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/http/middleware"
)

type APIService struct {
	engine  *gin.Engine
	context *appcontext.Context
}

func NewHTTPService(ctx *appcontext.Context) *APIService {
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
	v1 := h.engine.Group("/api/v1")
	h.setupAuthRoutes(v1)
	h.setupProjectRoutes(v1)
	h.setupDatasetRoutes(v1)
	h.setupTableRoutes(v1)
	h.setupColumnRoutes(v1)
	h.setupFileRoutes(v1)

	h.engine.Static("/static", "./static")
	h.engine.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})
}

func (h *APIService) setupAuthRoutes(group *gin.RouterGroup) {
	auth := group.Group("/auth")

	auth.GET("/login", Login(h.context))
	auth.GET("/callback", Callback(h.context))
	auth.POST("/logout", Logout(h.context))
	auth.GET("/me", middleware.JWTAuthMiddleware(), GetUserInfo(h.context))
}

func (h *APIService) setupProjectRoutes(group *gin.RouterGroup) {}

func (h *APIService) setupDatasetRoutes(group *gin.RouterGroup) {
	datasets := group.Group("/datasets")
	datasets.Use(middleware.JWTAuthMiddleware())

	datasets.GET("/", FetchDatasets(h.context))
}

func (h *APIService) setupTableRoutes(group *gin.RouterGroup) {}

func (h *APIService) setupColumnRoutes(group *gin.RouterGroup) {}

func (h *APIService) setupFileRoutes(group *gin.RouterGroup) {
	files := group.Group("/files")
	files.Use(middleware.JWTAuthMiddleware())

	files.POST("/", UploadFile(h.context))
}
