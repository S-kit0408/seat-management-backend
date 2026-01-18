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
		&entity.ReservationSettings{},
		&entity.FloorOperationHours{},
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

	// 予約設定の制約
	if err := createReservationSettingsConstraints(db); err != nil {
		return err
	}

	// フロア営業時間の制約
	if err := createFloorOperationHoursConstraints(db); err != nil {
		return err
	}

	// デフォルト予約設定のシード
	if err := seedDefaultReservationSettings(db); err != nil {
		log.Printf("Warning: Failed to seed default reservation settings: %v", err)
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

// Create constraints for reservation_settings table
func createReservationSettingsConstraints(db *gorm.DB) error {
	constraints := []string{
		// Unique index for active settings (only one can be active)
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_reservation_settings_active
           ON reservation_settings(is_active)
           WHERE is_active = true;`,

		// Validation constraints
		`ALTER TABLE reservation_settings
           DROP CONSTRAINT IF EXISTS chk_settings_min_lt_max_reservation;
           ALTER TABLE reservation_settings
           ADD CONSTRAINT chk_settings_min_lt_max_reservation
           CHECK (max_reservation_minutes > min_reservation_minutes);`,

		`ALTER TABLE reservation_settings
           DROP CONSTRAINT IF EXISTS chk_settings_positive_values;
           ALTER TABLE reservation_settings
           ADD CONSTRAINT chk_settings_positive_values
           CHECK (
               min_reservation_minutes > 0 AND
               max_reservation_minutes > 0 AND
               max_advance_booking_days > 0 AND
               check_in_minutes_before_start >= 0 AND
               check_in_grace_period_minutes >= 0 AND
               cancellation_deadline_minutes >= 0 AND
               max_extension_minutes > 0 AND
               max_extension_count >= 0
           );`,
	}

	for _, constraint := range constraints {
		if err := db.Exec(constraint).Error; err != nil {
			log.Printf("Warning: Failed to create reservation settings constraint: %v", err)
		}
	}

	log.Println("Reservation settings constraints created successfully")
	return nil
}

// seedDefaultReservationSettings creates default settings if none exist
func seedDefaultReservationSettings(db *gorm.DB) error {
	var count int64
	if err := db.Model(&entity.ReservationSettings{}).Count(&count).Error; err != nil {
		return err
	}

	// Only seed if no settings exist
	if count == 0 {
		defaultSettings := entity.GetDefaultSettings()
		if err := db.Create(defaultSettings).Error; err != nil {
			return err
		}
		log.Println("Default reservation settings seeded successfully")
	}

	return nil
}

// createFloorOperationHoursConstraints creates constraints for floor_operation_hours table
func createFloorOperationHoursConstraints(db *gorm.DB) error {
	constraints := []string{
		// Composite unique index: prevent duplicate (floor_id, day_of_week)
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_floor_operation_hours_unique
           ON floor_operation_hours(floor_id, day_of_week)
           WHERE deleted_at IS NULL;`,

		// Check constraint: open_time < close_time when not closed
		`ALTER TABLE floor_operation_hours
           DROP CONSTRAINT IF EXISTS chk_operation_hours_time_order;
           ALTER TABLE floor_operation_hours
           ADD CONSTRAINT chk_operation_hours_time_order
           CHECK (is_closed = true OR close_time > open_time);`,

		// Index for querying by floor
		`CREATE INDEX IF NOT EXISTS idx_floor_operation_hours_floor
           ON floor_operation_hours(floor_id)
           WHERE deleted_at IS NULL;`,

		// Index for querying by day of week
		`CREATE INDEX IF NOT EXISTS idx_floor_operation_hours_day
           ON floor_operation_hours(day_of_week)
           WHERE deleted_at IS NULL;`,
	}

	for _, constraint := range constraints {
		if err := db.Exec(constraint).Error; err != nil {
			log.Printf("Warning: Failed to create floor operation hours constraint: %v", err)
			// Continue on error (constraint may already exist)
		}
	}

	log.Println("Floor operation hours constraints created successfully")
	return nil
}
