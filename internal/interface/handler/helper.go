package handler

import (
	"github.com/gin-gonic/gin"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/middleware"
	"seat-management-backend/internal/usecase"
)

func GetAuthenticatedUser(c *gin.Context, userUsecase usecase.UserUsecase) (*entity.User, error) {
	clerkUserID, err := middleware.GetClerkUserID(c)
	if err != nil {
		return nil, err
	}

	user, err := userUsecase.GetByClerkUserID(c.Request.Context(), clerkUserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
