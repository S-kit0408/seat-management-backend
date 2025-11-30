package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	svix "github.com/svix/svix-webhooks/go"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/usecase"
)

type WebhookHandler struct {
	userUsecase usecase.UserUsecase
}

func NewWebhookHandler(uu usecase.UserUsecase) *WebhookHandler {
	return &WebhookHandler{
		userUsecase: uu,
	}
}

// WebhookEvent はClerkからのWebhookイベントの構造
type WebhookEvent struct {
	Type   string          `json:"type"`
	Object string          `json:"object"`
	Data   json.RawMessage `json:"data"`
}

// ClerkUserData はClerkユーザーデータの構造
type ClerkUserData struct {
	ID               string                 `json:"id"`
	EmailAddresses   []ClerkEmailAddress    `json:"email_addresses"`
	FirstName        *string                `json:"first_name"`
	LastName         *string                `json:"last_name"`
	ImageURL         *string                `json:"image_url"`
	ExternalAccounts []ClerkExternalAccount `json:"external_accounts"`
	PasswordEnabled  bool                   `json:"password_enabled"`
}

type ClerkEmailAddress struct {
	EmailAddress string `json:"email_address"`
}

type ClerkExternalAccount struct {
	Provider     string `json:"provider"`
	EmailAddress string `json:"email_address"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	AvatarURL    string `json:"avatar_url"`
}

// HandleClerkWebhook はClerkからのWebhookを処理
func (h *WebhookHandler) HandleClerkWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[Webhook] Failed to read body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストボディの読み取りに失敗しました"})
		return
	}

	if os.Getenv("GIN_MODE") == "debug" {
		log.Printf("[Webhook] Received payload: %s", string(payload))
	}

	headers := http.Header{}
	headers.Set("svix-id", c.GetHeader("svix-id"))
	headers.Set("svix-timestamp", c.GetHeader("svix-timestamp"))
	headers.Set("svix-signature", c.GetHeader("svix-signature"))

	webhookSecret := os.Getenv("CLERK_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Printf("[Webhook] CLERK_WEBHOOK_SECRET is not set")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Webhook設定エラー"})
		return
	}

	wh, err := svix.NewWebhook(webhookSecret)
	if err != nil {
		log.Printf("[Webhook] Failed to initialize webhook: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Webhook初期化に失敗しました"})
		return
	}

	var evt WebhookEvent
	err = wh.Verify(payload, headers)
	if err != nil {
		log.Printf("[Webhook] Signature verification failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Webhook署名の検証に失敗しました"})
		return
	}

	if err := json.Unmarshal(payload, &evt); err != nil {
		log.Printf("[Webhook] Failed to parse event: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "イベントのパースに失敗しました"})
		return
	}

	log.Printf("[Webhook] Event type: %s", evt.Type)

	switch evt.Type {
	case "user.created":
		if err := h.handleUserCreated(c, evt.Data); err != nil {
			log.Printf("[Webhook] user.created handler error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "イベントを受信しましたが、処理中にエラーが発生しました",
				"error":   err.Error(),
			})
			return
		}
	case "user.updated":
		if err := h.handleUserUpdated(c, evt.Data); err != nil {
			log.Printf("[Webhook] user.updated handler error: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "イベントを受信しましたが、処理中にエラーが発生しました",
				"error":   err.Error(),
			})
			return
		}
	case "user.deleted":
		if err := h.handleUserDeleted(c, evt.Data); err != nil {
			log.Printf("[Webhook] user.deleted handler error: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "イベントを受信しましたが、処理中にエラーが発生しました",
				"error":   err.Error(),
			})
			return
		}
	default:
		log.Printf("[Webhook] Unhandled event type: %s", evt.Type)
		c.JSON(http.StatusOK, gin.H{"message": "処理対象外のイベントタイプです"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhookの処理が完了しました"})
}

// 認証プロバイダーを判定する
func determineAuthProvider(clerkUser ClerkUserData) entity.AuthProvider {
	// デバッグログを追加
	log.Printf("[Webhook] Determining auth provider...")
	log.Printf("[Webhook] External accounts count: %d", len(clerkUser.ExternalAccounts))
	log.Printf("[Webhook] Password enabled: %v", clerkUser.PasswordEnabled)

	// 外部アカウント（Google）が存在する場合
	if len(clerkUser.ExternalAccounts) > 0 {
		provider := clerkUser.ExternalAccounts[0].Provider
		log.Printf("[Webhook] External account provider: %s", provider)

		switch provider {
		case "oauth_google":
			log.Printf("[Webhook] Detected Google OAuth")
			return entity.AuthProviderGoogle
		default:
			log.Printf("[Webhook] Unknown OAuth provider: %s", provider)
			return entity.AuthProviderUnknown
		}
	}

	// パスワード認証が有効な場合
	if clerkUser.PasswordEnabled {
		log.Printf("[Webhook] Detected password-based authentication")
		return entity.AuthProviderEmail
	}

	log.Printf("[Webhook] Could not determine auth provider, returning unknown")
	return entity.AuthProviderUnknown
}

func (h *WebhookHandler) handleUserCreated(c *gin.Context, data json.RawMessage) error {
	var clerkUser ClerkUserData
	if err := json.Unmarshal(data, &clerkUser); err != nil {
		return fmt.Errorf("ユーザーデータのパース失敗: %w", err)
	}

	log.Printf("[Webhook] Processing user.created for Clerk ID: %s", clerkUser.ID)

	if len(clerkUser.EmailAddresses) == 0 {
		log.Printf("[Webhook] No email addresses found for user %s (test event?)", clerkUser.ID)
		return fmt.Errorf("メールアドレスが見つかりません（テストイベントの可能性があります）")
	}

	email := clerkUser.EmailAddresses[0].EmailAddress
	if email == "" {
		return fmt.Errorf("メールアドレスが空です")
	}

	name := ""
	if clerkUser.FirstName != nil && clerkUser.LastName != nil {
		name = fmt.Sprintf("%s %s", *clerkUser.FirstName, *clerkUser.LastName)
	} else if clerkUser.FirstName != nil {
		name = *clerkUser.FirstName
	} else if clerkUser.LastName != nil {
		name = *clerkUser.LastName
	} else {
		name = strings.Split(email, "@")[0]
	}

	existingUser, err := h.userUsecase.GetByClerkUserID(c.Request.Context(), clerkUser.ID)
	if err == nil && existingUser != nil {
		log.Printf("[Webhook] User already exists: %s", clerkUser.ID)
		return nil
	}

	// ⭐ 認証プロバイダーを判定
	authProvider := determineAuthProvider(clerkUser)
	log.Printf("[Webhook] Determined auth provider: %s", authProvider)

	user := &entity.User{
		ClerkUserID:           clerkUser.ID,
		Email:                 email,
		Name:                  name,
		AvatarURL:             clerkUser.ImageURL,
		DefaultPrivacySetting: entity.PrivacyPrivate,
		PrimaryAuthProvider:   authProvider,
	}

	if err := h.userUsecase.Create(c.Request.Context(), user); err != nil {
		return fmt.Errorf("ユーザーの作成失敗: %w", err)
	}

	log.Printf("[Webhook] User created successfully: %s (%s) with provider: %s", clerkUser.ID, email, authProvider)
	return nil
}

func (h *WebhookHandler) handleUserUpdated(c *gin.Context, data json.RawMessage) error {
	var clerkUser ClerkUserData
	if err := json.Unmarshal(data, &clerkUser); err != nil {
		return fmt.Errorf("ユーザーデータのパース失敗: %w", err)
	}

	log.Printf("[Webhook] Processing user.updated for Clerk ID: %s", clerkUser.ID)

	user, err := h.userUsecase.GetByClerkUserID(c.Request.Context(), clerkUser.ID)
	if err != nil {
		return fmt.Errorf("ユーザーが見つかりません: %w", err)
	}

	if len(clerkUser.EmailAddresses) > 0 {
		user.Email = clerkUser.EmailAddresses[0].EmailAddress
	}

	name := ""
	if clerkUser.FirstName != nil && clerkUser.LastName != nil {
		name = fmt.Sprintf("%s %s", *clerkUser.FirstName, *clerkUser.LastName)
	} else if clerkUser.FirstName != nil {
		name = *clerkUser.FirstName
	} else if clerkUser.LastName != nil {
		name = *clerkUser.LastName
	}
	if name != "" {
		user.Name = name
	}

	user.AvatarURL = clerkUser.ImageURL
	user.PrimaryAuthProvider = determineAuthProvider(clerkUser)

	if err := h.userUsecase.Update(c.Request.Context(), user); err != nil {
		return fmt.Errorf("ユーザーの更新失敗: %w", err)
	}

	log.Printf("[Webhook] User updated successfully: %s", clerkUser.ID)
	return nil
}

func (h *WebhookHandler) handleUserDeleted(c *gin.Context, data json.RawMessage) error {
	var clerkUser ClerkUserData
	if err := json.Unmarshal(data, &clerkUser); err != nil {
		return fmt.Errorf("ユーザーデータのパース失敗: %w", err)
	}

	log.Printf("[Webhook] Processing user.deleted for Clerk ID: %s", clerkUser.ID)

	user, err := h.userUsecase.GetByClerkUserID(c.Request.Context(), clerkUser.ID)
	if err != nil {
		return fmt.Errorf("ユーザーが見つかりません: %w", err)
	}

	if err := h.userUsecase.Delete(c.Request.Context(), user.ID); err != nil {
		return fmt.Errorf("ユーザーの削除失敗: %w", err)
	}

	log.Printf("[Webhook] User deleted successfully: %s", clerkUser.ID)
	return nil
}

// RegisterRoutes はWebhookルートを登録
func (h *WebhookHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/webhooks/clerk", h.HandleClerkWebhook)
}
