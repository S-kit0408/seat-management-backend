package database

import (
	"log"

	"gorm.io/gorm"
	"seat-management-backend/internal/domain/entity"
)

// AutoMigrate はテーブルとENUM型を作成
func AutoMigrate(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// ENUM型を作成（存在しない場合のみ）
	if err := createEnums(db); err != nil {
		return err
	}

	// テーブルを作成
	err := db.AutoMigrate(
		&entity.User{},
	)

	if err != nil {
		return err
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// createEnums はENUM型を作成
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
	}

	for _, enum := range enums {
		if err := db.Exec(enum).Error; err != nil {
			return err
		}
	}

	log.Println("ENUM types created successfully")
	return nil
}
