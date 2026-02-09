package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "seat-management-backend/docs"
	"seat-management-backend/internal/infrastructure/ai"
	"seat-management-backend/internal/infrastructure/persistence"
	"seat-management-backend/internal/infrastructure/websocket"
	"seat-management-backend/internal/interface/handler"
	"seat-management-backend/internal/middleware"
	"seat-management-backend/internal/usecase"
	"seat-management-backend/pkg/database"
)

// @title Seat Management Backend API
// @version 1.0
// @description AI-powered seat reservation system API
// @termsOfService http://swagger.io/terms/
// @contact.name Support
// @contact.url http://localhost:8080/support
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @basePath /api
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token
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
	reservationSettingsRepo := persistence.NewReservationSettingsRepository(db)
	floorOperationHoursRepo := persistence.NewFloorOperationHoursRepository(db)

	// Initialize WebSocket Hub
	wsHub := websocket.NewHub()

	// Start WebSocket Hub in background
	wsCtx, wsCancel := context.WithCancel(context.Background())
	defer wsCancel()
	go wsHub.Run(wsCtx)

	// Initialize EventNotifier
	eventNotifier := usecase.NewEventNotifier(wsHub)

	userUsecase := usecase.NewUserUsecase(userRepo)
	friendUsecase := usecase.NewFriendUsecase(friendRequestRepo, friendshipRepo, userRepo, db)
	floorUsecase := usecase.NewFloorUsecase(floorRepo)
	seatUsecase := usecase.NewSeatUsecase(seatRepo, eventNotifier)
	reservationSettingsUsecase := usecase.NewReservationSettingsUsecase(
		reservationSettingsRepo,
		db,
	)
	reservationUsecase := usecase.NewReservationUsecase(
		reservationRepo,
		recurringReservationRepo,
		userRepo,
		seatRepo,
		friendshipRepo,
		reservationSettingsRepo,
		floorOperationHoursRepo,
		db,
		eventNotifier,
	)
	recurringReservationUsecase := usecase.NewRecurringReservationUsecase(
		recurringReservationRepo,
		reservationRepo,
		userRepo,
		seatRepo,
		reservationSettingsRepo,
		floorOperationHoursRepo,
		db,
	)
	floorOperationHoursUsecase := usecase.NewFloorOperationHoursUsecase(
		floorOperationHoursRepo,
		floorRepo,
		db,
	)

	// Gemini AI 検索サービス初期化
	geminiService, err := ai.NewGeminiService(context.Background())
	if err != nil {
		log.Printf("Warning: Failed to initialize Gemini service: %v", err)
	}
	defer func() {
		if geminiService != nil {
			geminiService.Close()
		}
	}()

	// AI検索 Usecase（Gemini初期化成功時のみ）
	var aiSearchUsecase usecase.AiSearchUsecase
	if geminiService != nil {
		aiSearchUsecase = usecase.NewAiSearchUsecase(geminiService, seatRepo)
	}

	// 管理者設定
	setupAdmins(userUsecase)

	// Start no-show batch processing in background
	batchCtx, batchCancel := context.WithCancel(context.Background())
	defer batchCancel()
	go startNoShowBatchJob(batchCtx, reservationUsecase)

	// handler init
	webhookHandler := handler.NewWebhookHandler(userUsecase)
	adminHandler := handler.NewAdminHandler(userUsecase, userRepo)
	userHandler := handler.NewUserHandler(userUsecase)
	friendHandler := handler.NewFriendHandler(friendUsecase, userUsecase)
	floorHandler := handler.NewFloorHandler(floorUsecase, userRepo)
	seatHandler := handler.NewSeatHandler(seatUsecase, userRepo)
	reservationHandler := handler.NewReservationHandler(reservationUsecase, userUsecase)
	recurringReservationHandler := handler.NewRecurringReservationHandler(recurringReservationUsecase, userUsecase, userRepo)
	reservationSettingsHandler := handler.NewReservationSettingsHandler(reservationSettingsUsecase, userRepo)
	floorOperationHoursHandler := handler.NewFloorOperationHoursHandler(floorOperationHoursUsecase, userRepo)
	wsHandler := handler.NewWebSocketHandler(wsHub, userUsecase, friendshipRepo)

	// Search handler (AI検索が有効な場合のみ）
	var searchHandler *handler.SearchHandler
	if aiSearchUsecase != nil {
		searchHandler = handler.NewSearchHandler(aiSearchUsecase)
	}

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

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ルートの登録
	webhookHandler.RegisterRoutes(r)
	adminHandler.RegisterRoutes(r)
	userHandler.RegisterRoutes(r)
	friendHandler.RegisterRoutes(r)
	floorHandler.RegisterRoutes(r)
	seatHandler.RegisterRoutes(r)
	reservationHandler.RegisterRoutes(r)
	recurringReservationHandler.RegisterRoutes(r)
	reservationSettingsHandler.RegisterRoutes(r)
	floorOperationHoursHandler.RegisterRoutes(r)
	wsHandler.RegisterRoutes(r)
	if searchHandler != nil {
		searchHandler.RegisterRoutes(r)
	}

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

// startNoShowBatchJob はno-show自動変更のバッチ処理を定期実行
// 毎分実行され、利用終了時刻を超過してチェックインされていない予約をno-showに変更
func startNoShowBatchJob(ctx context.Context, reservationUsecase usecase.ReservationUsecase) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("No-show batch job started (runs every 1 minute)")

	for {
		select {
		case <-ticker.C:
			// バッチ処理を実行
			if err := reservationUsecase.MarkPastReservationsAsNoShow(ctx); err != nil {
				log.Printf("[ERROR] No-show batch job failed: %v", err)
			}

		case <-ctx.Done():
			log.Println("No-show batch job stopped")
			return
		}
	}
}
