package usecase

import (
	"context"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
)

// ユーザー関連のビジネスロジックを定義
type UserUsecase interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id string) (*entity.User, error)
	GetByClerkUserID(ctx context.Context, clerkUserID string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	UpdateLastLogin(ctx context.Context, userID string) error
	UpdateRole(ctx context.Context, userID string, role entity.UserRole) error
	SetupAdminUsers(ctx context.Context, adminEmails []string) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*entity.User, error)
}

type userUsecase struct {
	userRepo repository.UserRepository
}

// UserUsecaseの新しいインスタンスを作成
func NewUserUsecase(ur repository.UserRepository) UserUsecase {
	return &userUsecase{
		userRepo: ur,
	}
}

// ユーザーを作成
func (u *userUsecase) Create(ctx context.Context, user *entity.User) error {
	// ビジネスロジック: デフォルト値の設定など
	if user.DefaultPrivacySetting == "" {
		user.DefaultPrivacySetting = entity.PrivacyPrivate
	}
	if user.PrimaryAuthProvider == "" {
		user.PrimaryAuthProvider = entity.AuthProviderUnknown
	}

	return u.userRepo.Create(ctx, user)
}

// IDでユーザーを取得
func (u *userUsecase) GetByID(ctx context.Context, id string) (*entity.User, error) {
	return u.userRepo.FindByID(ctx, id)
}

// Clerk User IDでユーザーを取得
func (u *userUsecase) GetByClerkUserID(ctx context.Context, clerkUserID string) (*entity.User, error) {
	return u.userRepo.FindByClerkUserID(ctx, clerkUserID)
}

// Emailでユーザーを取得
func (u *userUsecase) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return u.userRepo.FindByEmail(ctx, email)
}

// ユーザー情報を更新
func (u *userUsecase) Update(ctx context.Context, user *entity.User) error {
	// ビジネスロジック: バリデーションなど
	if user.Email == "" {
		return entity.ErrInvalidEmail
	}
	if user.Name == "" {
		return entity.ErrInvalidName
	}

	return u.userRepo.Update(ctx, user)
}

// 最終ログイン時刻を更新
func (u *userUsecase) UpdateLastLogin(ctx context.Context, userID string) error {
	return u.userRepo.UpdateLastLogin(ctx, userID)
}

// ユーザーを削除（ソフトデリート）
func (u *userUsecase) Delete(ctx context.Context, id string) error {
	return u.userRepo.Delete(ctx, id)
}

// ユーザー一覧を取得
func (u *userUsecase) List(ctx context.Context, limit, offset int) ([]*entity.User, error) {
	// ビジネスロジック: limitの最大値チェックなど
	if limit <= 0 || limit > 100 {
		limit = 20 // デフォルト値
	}
	if offset < 0 {
		offset = 0
	}

	return u.userRepo.List(ctx, limit, offset)
}

// ユーザーのロールを更新
func (u *userUsecase) UpdateRole(ctx context.Context, userID string, role entity.UserRole) error {
	// ビジネスロジック: ロールのバリデーション
	if !role.IsValid() {
		return entity.ErrInvalidRole
	}

	return u.userRepo.UpdateRole(ctx, userID, role)
}

// 環境変数で指定された管理者を設定
func (u *userUsecase) SetupAdminUsers(ctx context.Context, adminEmails []string) error {
	// 全ユーザーを取得（limit=100で十分）
	allUsers, err := u.userRepo.List(ctx, 100, 0)
	if err != nil {
		return err
	}

	// adminEmailsをマップに変換（高速検索用）
	adminEmailMap := make(map[string]bool)
	for _, email := range adminEmails {
		adminEmailMap[email] = true
	}

	// 全ユーザーをチェック
	for _, user := range allUsers {
		shouldBeAdmin := adminEmailMap[user.Email]

		// 現在のロールと期待されるロールが異なる場合のみ更新
		if shouldBeAdmin && user.Role != entity.RoleAdmin {
			// userをadminに昇格
			if err := u.userRepo.UpdateRole(ctx, user.ID, entity.RoleAdmin); err != nil {
				return err
			}
		} else if !shouldBeAdmin && user.Role == entity.RoleAdmin {
			// adminをuserに降格
			if err := u.userRepo.UpdateRole(ctx, user.ID, entity.RoleUser); err != nil {
				return err
			}
		}
	}

	return nil
}
