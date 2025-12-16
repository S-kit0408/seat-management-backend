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
		&entity.Reservation{},
		&entity.RecurringReservation{},
	)

	if err != nil {
		return err
	}

	// index制約
	if err := createFriendshipConstraints(db); err != nil {
		return err
	}

	// 予約関連の制約とインデックス（追加）
	if err := createReservationConstraints(db); err != nil {
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

		// 予約タイプのENUM（追加）
		`DO $$ BEGIN
		    CREATE TYPE reservation_type_enum AS ENUM('instant', 'scheduled', 'recurring');
	    EXCEPTION
		    WHEN duplicate_object THEN null;
	    END $$;`,

		// 予約ステータスのENUM（追加）
		`DO $$ BEGIN
			CREATE TYPE reservation_status_enum AS ENUM('reserved', 'in_use', 'completed', 'cancelled', 'no_show');
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

// 予約テーブル用の制約とインデックスを作成
func createReservationConstraints(db *gorm.DB) error {
	constraints := []string{
		// btree_gist拡張を有効化（時間範囲の重複チェックに必要）
		`CREATE EXTENSION IF NOT EXISTS btree_gist;`,

		// 予約時間の整合性チェック
		`ALTER TABLE reservations 
           DROP CONSTRAINT IF EXISTS chk_reservation_time_order;
           ALTER TABLE reservations 
           ADD CONSTRAINT chk_reservation_time_order 
           CHECK (end_time > start_time);`,

		// チェックアウトがチェックインより後であることを保証
		`ALTER TABLE reservations 
           DROP CONSTRAINT IF EXISTS chk_checkout_after_checkin;
           ALTER TABLE reservations 
           ADD CONSTRAINT chk_checkout_after_checkin 
           CHECK (checked_out_at IS NULL OR checked_in_at IS NULL OR checked_out_at > checked_in_at);`,

		// 座席の予約重複を防ぐ（除外制約）
		`ALTER TABLE reservations 
           DROP CONSTRAINT IF EXISTS reservations_no_overlap;
           ALTER TABLE reservations 
           ADD CONSTRAINT reservations_no_overlap 
           EXCLUDE USING gist (
             seat_id WITH =,
             tstzrange(start_time, end_time) WITH &&
           ) WHERE (deleted_at IS NULL AND status IN ('reserved', 'in_use'));`,

		// 複合インデックス
		`CREATE INDEX IF NOT EXISTS idx_reservations_user_status 
           ON reservations(user_id, status) 
           WHERE deleted_at IS NULL;`,

		`CREATE INDEX IF NOT EXISTS idx_reservations_seat_time 
           ON reservations(seat_id, start_time, end_time) 
           WHERE deleted_at IS NULL AND status IN ('reserved', 'in_use');`,

		`CREATE INDEX IF NOT EXISTS idx_reservations_time_range 
           ON reservations(start_time, end_time) 
           WHERE deleted_at IS NULL;`,

		// 自動キャンセル処理用インデックス
		`CREATE INDEX IF NOT EXISTS idx_reservations_auto_cancel 
           ON reservations(status, start_time, checked_in_at) 
           WHERE deleted_at IS NULL AND status = 'reserved' AND checked_in_at IS NULL;`,

		// 定期予約の時刻順序チェック
		`ALTER TABLE recurring_reservations 
           DROP CONSTRAINT IF EXISTS chk_recurring_time_order;
           ALTER TABLE recurring_reservations 
           ADD CONSTRAINT chk_recurring_time_order 
           CHECK (end_time > start_time);`,
	}

	for _, constraint := range constraints {
		if err := db.Exec(constraint).Error; err != nil {
			log.Printf("Warning: Failed to create constraint: %v", err)
			// 制約作成の失敗は警告のみで続行
		}
	}

	log.Println("Reservation constraints created successfully")
	return nil
}
