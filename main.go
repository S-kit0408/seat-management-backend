package main

import (
	"log"
	"os"
	"seat-management-backend/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"seat-management-backend/internal/infrastructure/persistence"
	"seat-management-backend/internal/interface/handler"
	"seat-management-backend/internal/usecase"
	"seat-management-backend/pkg/database"
)

func main() {
	// 環境変数の読み込み
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Clerk SDKを初期化
	if err := middleware.InitClerk(); err != nil {
		log.Fatalln("Failed to initialize Clerk:", err)
	}

	// データベース接続
	db, err := database.NewPostgresDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// migration
	if err := database.AutoMigrate(db); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// 依存関係の注入
	userRepo := persistence.NewUserRepository(db)
	userUsecase := usecase.NewUserUsecase(userRepo)

	// ハンドラーの初期化
	userHandler := handler.NewUserHandler(userUsecase)
	webhookHandler := handler.NewWebhookHandler(userUsecase)

	// Ginルーターの初期化
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
	userHandler.RegisterRoutes(r)
	webhookHandler.RegisterRoutes(r)

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
