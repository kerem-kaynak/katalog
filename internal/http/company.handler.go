package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kerem-kaynak/katalog/internal/appcontext"
	"github.com/kerem-kaynak/katalog/internal/entity"
	"github.com/kerem-kaynak/katalog/internal/utils"
	"go.uber.org/zap"
)

func CreateCompany(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		type createCompanyRequest struct {
			CompanyName string `json:"companyName" binding:"required"`
			ProjectName string `json:"projectName" binding:"required"`
		}

		var request createCompanyRequest

		if err := c.BindJSON(&request); err != nil {
			ctx.Logger.Error("Failed to bind request", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to bind request"})
			return
		}

		userID, err := utils.GetUserIDFromClaims(c)
		if err != nil {
			ctx.Logger.Error("Failed to get user ID from claims", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		company := entity.Company{
			Name: request.CompanyName,
		}

		if err := ctx.DB.Create(&company).Error; err != nil {
			ctx.Logger.Error("Failed to create company", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create company"})
			return
		}

		if err := ctx.DB.Model(&entity.User{}).Where("id = ?", userID).Update("company_id", company.ID).Error; err != nil {
			ctx.Logger.Error("Failed to update user's company ID", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user's company ID"})
			return
		}

		project := entity.Project{
			Name:      request.ProjectName,
			CompanyID: company.ID,
		}

		if err := ctx.DB.Create(&project).Error; err != nil {
			ctx.Logger.Error("Failed to create project", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		}

		c.JSON(http.StatusOK, gin.H{"message": "Company and project created successfully", "company_id": company.ID, "project_id": project.ID})
	}
}

func GetCompanyMembers(ctx *appcontext.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserIDFromClaims(c)
		if err != nil {
			ctx.Logger.Error("Failed to get user ID from claims", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID from claims"})
			return
		}

		var user entity.User
		if err := ctx.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			ctx.Logger.Error("Failed to get user by ID", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user by ID"})
			return
		}

		var companyMembers []entity.User
		if err := ctx.DB.Where("company_id = ? AND status = ?", user.CompanyID, "active").Find(&companyMembers).Error; err != nil {
			ctx.Logger.Error("Failed to get team members from database", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get team members from database"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"members": companyMembers})
	}
}
