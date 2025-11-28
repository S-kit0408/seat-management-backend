package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/gin-gonic/gin"
)

// InitClerk はClerkクライアントを初期化
func InitClerk() error {
	secretKey := os.Getenv("CLERK_SECRET_KEY")
	if secretKey == "" {
		return errors.New("CLERK_SECRET_KEY is not set")
	}
	clerk.SetKey(secretKey)
	return nil
}

// ClerkAuthMiddleware はClerkトークンを検証するミドルウェア
func ClerkAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Authorizationヘッダーからトークンを取得
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
			c.Abort()
			return
		}

		// "Bearer "プレフィックスを削除
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "無効な認証形式です"})
			c.Abort()
			return
		}

		// デバッグログ（開発時のみ）
		if os.Getenv("GIN_MODE") == "debug" {
			tokenPreview := tokenString
			if len(tokenString) > 20 {
				tokenPreview = tokenString[:20] + "..."
			}
			fmt.Printf("[DEBUG] Token received: %s\n", tokenPreview)
		}

		// セッショントークンを検証
		ctx := context.Background()

		claims, err := jwt.Verify(ctx, &jwt.VerifyParams{
			Token: tokenString,
		})

		if err != nil {
			// デバッグログ
			fmt.Printf("[ERROR] Token verification failed: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "無効なトークンです",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// デバッグログ
		if os.Getenv("GIN_MODE") == "debug" {
			fmt.Printf("[DEBUG] Token verified successfully. Subject: %s\n", claims.Subject)
		}

		// コンテキストにユーザー情報を設定
		c.Set("clerkUserID", claims.Subject)

		// セッションIDがあれば設定
		if claims.SessionID != "" {
			c.Set("sessionID", claims.SessionID)
		}

		// 組織情報があれば設定
		if claims.ActiveOrganizationID != "" {
			c.Set("organizationID", claims.ActiveOrganizationID)
		}
		if claims.ActiveOrganizationRole != "" {
			c.Set("organizationRole", claims.ActiveOrganizationRole)
		}

		c.Next()
	}
}

// GetClerkUserID はコンテキストからClerk User IDを取得
func GetClerkUserID(c *gin.Context) (string, error) {
	userID, exists := c.Get("clerkUserID")
	if !exists {
		return "", errors.New("ユーザーIDが見つかりません")
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return "", errors.New("無効なユーザーID形式です")
	}

	return userIDStr, nil
}

// GetSessionID はコンテキストからSession IDを取得
func GetSessionID(c *gin.Context) (string, bool) {
	sessionID, exists := c.Get("sessionID")
	if !exists {
		return "", false
	}
	return sessionID.(string), true
}

// GetEmail はコンテキストからEmailを取得
func GetEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("email")
	if !exists {
		return "", false
	}
	return email.(string), true
}

// GetName はコンテキストからNameを取得
func GetName(c *gin.Context) (string, bool) {
	name, exists := c.Get("name")
	if !exists {
		return "", false
	}
	return name.(string), true
}
