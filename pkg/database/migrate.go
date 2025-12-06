package database

import (
	"log"

	"seat-management-backend/internal/domain/entity"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// ENUM型を作成（存在しない場合のみ）
	if err := createEnums(db); err != nil {
		return err
	}

	// テーブルを作成
	err := db.AutoMigrate(
		&entity.User{},
		&entity.FriendRequest{},
		&entity.Friendship{},
		&entity.Floor{},
		&entity.Seat{},
	)

	if err != nil {
		return err
	}

	// index制約
	if err := createFriendshipConstraints(db); err != nil {
		return err
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// ENUM型を作成
func createEnums(db *gorm.DB) error {
	enums := []string{
		`DO $$ BEGIN
            CREATE TYPE privacy_setting_enum AS ENUM('public', 'friends', 'private');
        EXCEPTION
            WHEN duplicate_object THEN null;
        END $$;`,
		`DO $$ BEGIN
            CREATE TYPE auth_provider_enum AS ENUM('email', 'google', 'unknown');
        EXCEPTION
            WHEN duplicate_object THEN null;
        END $$;`,
		`DO $$ BEGIN
            CREATE TYPE user_role_enum AS ENUM ('user', 'admin', 'moderator');
        EXCEPTION
            WHEN duplicate_object THEN null;
        END $$;`,
		`DO $$ BEGIN
			CREATE TYPE request_status_enum AS ENUM('pending', 'accepted', 'rejected', 'cancelled');
		EXCEPTION
			WHEN duplicate_object THEN null;
  		END $$;`,
	}

	for _, enum := range enums {
		if err := db.Exec(enum).Error; err != nil {
			return err
		}
	}

	log.Println("ENUM types created successfully")
	return nil
}

// フレンドシップテーブル用の制約とインデックスを作成
func createFriendshipConstraints(db *gorm.DB) error {
	constraints := []string{
		// friend_requests: 同じユーザー間の重複申請を防ぐ
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_friend_requests_unique 
                 ON friend_requests(requester_id, addressee_id);`,

		// 自分自身への申請を防ぐ
		`ALTER TABLE friend_requests 
                 DROP CONSTRAINT IF EXISTS chk_friend_requests_no_self;
                 ALTER TABLE friend_requests 
                 ADD CONSTRAINT chk_friend_requests_no_self 
                 CHECK (requester_id != addressee_id);`,

		// 保留中の申請を高速検索するための部分インデックス
		`CREATE INDEX IF NOT EXISTS idx_friend_requests_pending 
                 ON friend_requests(addressee_id) 
                 WHERE status = 'pending';`,

		// 同じユーザー間の重複フレンド関係を防ぐ
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_friendships_unique 
                 ON friendships(user_id1, user_id2);`,

		// user_id1が常にuser_id2より小さいことを保証
		`ALTER TABLE friendships 
                 DROP CONSTRAINT IF EXISTS chk_friendships_ordered;
                 ALTER TABLE friendships 
                 ADD CONSTRAINT chk_friendships_ordered 
                 CHECK (user_id1 < user_id2);`,

		// 自分自身とのフレンド関係を防ぐ
		`ALTER TABLE friendships 
                 DROP CONSTRAINT IF EXISTS chk_friendships_no_self;
                 ALTER TABLE friendships 
                 ADD CONSTRAINT chk_friendships_no_self 
                 CHECK (user_id1 != user_id2);`,
	}

	for _, constraint := range constraints {
		if err := db.Exec(constraint).Error; err != nil {
			log.Printf("Warning: Failed to create constraint: %v", err)
			// 制約作成の失敗は警告のみで続行
		}
	}

	log.Println("Friendship constraints created successfully")
	return nil
}
