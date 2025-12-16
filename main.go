package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"seat-management-backend/internal/infrastructure/persistence"
	"seat-management-backend/internal/interface/handler"
	"seat-management-backend/internal/middleware"
	"seat-management-backend/internal/usecase"
	"seat-management-backend/pkg/database"
)

func main() {
	// 環境変数読み込み
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Clerk SDK init
	if err := middleware.InitClerk(); err != nil {
		log.Fatalln("Failed to initialize Clerk:", err)
	}

	// connect db
	db, err := database.NewPostgresDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// migration
	if err := database.AutoMigrate(db); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// 依存関係
	userRepo := persistence.NewUserRepository(db)
	friendRequestRepo := persistence.NewFriendRequestRepository(db)
	friendshipRepo := persistence.NewFriendshipRepository(db)
	floorRepo := persistence.NewFloorRepository(db)
	seatRepo := persistence.NewSeatRepository(db)
	reservationRepo := persistence.NewReservationRepository(db)
	recurringReservationRepo := persistence.NewRecurringReservationRepository(db)

	userUsecase := usecase.NewUserUsecase(userRepo)
	friendUsecase := usecase.NewFriendUsecase(friendRequestRepo, friendshipRepo, userRepo, db)
	floorUsecase := usecase.NewFloorUsecase(floorRepo)
	seatUsecase := usecase.NewSeatUsecase(seatRepo)
	reservationUsecase := usecase.NewReservationUsecase(
		reservationRepo,
		recurringReservationRepo,
		userRepo,
		seatRepo,
		friendshipRepo,
		db,
	)
	recurringReservationUsecase := usecase.NewRecurringReservationUsecase(
		recurringReservationRepo,
		reservationRepo,
		userRepo,
		seatRepo,
		db,
	)

	// 管理者設定
	setupAdmins(userUsecase)

	// handler init
	webhookHandler := handler.NewWebhookHandler(userUsecase)
	adminHandler := handler.NewAdminHandler(userUsecase, userRepo)
	userHandler := handler.NewUserHandler(userUsecase)
	friendHandler := handler.NewFriendHandler(friendUsecase, userUsecase)
	floorHandler := handler.NewFloorHandler(floorUsecase, userRepo)
	seatHandler := handler.NewSeatHandler(seatUsecase, userRepo)
	reservationHandler := handler.NewReservationHandler(reservationUsecase, userUsecase)
	recurringReservationHandler := handler.NewRecurringReservationHandler(recurringReservationUsecase, userUsecase, userRepo)

	// ルーターの初期化
	r := gin.Default()

	// CORS設定
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{os.Getenv("FRONTEND_URL")},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// ヘルスチェックエンドポイント
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":   "ok",
			"database": "connected",
		})
	})

	// ルートの登録
	webhookHandler.RegisterRoutes(r)
	adminHandler.RegisterRoutes(r)
	userHandler.RegisterRoutes(r)
	friendHandler.RegisterRoutes(r)
	floorHandler.RegisterRoutes(r)
	seatHandler.RegisterRoutes(r)
	reservationHandler.RegisterRoutes(r)
	recurringReservationHandler.RegisterRoutes(r)

	// サーバー起動
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// setupAdmins は環境変数で指定された管理者を設定
func setupAdmins(userUsecase usecase.UserUsecase) {
	adminEmailsStr := os.Getenv("ADMIN_EMAILS")
	if adminEmailsStr == "" {
		log.Println("No admin emails configured")
		return
	}

	// カンマ区切りでメールアドレスを分割
	adminEmails := strings.Split(adminEmailsStr, ",")
	for i, email := range adminEmails {
		adminEmails[i] = strings.TrimSpace(email)
	}

	ctx := context.Background()
	if err := userUsecase.SetupAdminUsers(ctx, adminEmails); err != nil {
		log.Printf("Warning: Failed to setup admin users: %v", err)
		return
	}

	log.Printf("Admin users configured: %v", adminEmails)
}
