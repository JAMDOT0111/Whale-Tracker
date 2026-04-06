package main

import (
	"context"
	"eth-sweeper/handler"
	"eth-sweeper/service"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	etherscanClient := service.NewEtherscanClient()
	graphService := service.NewGraphService(etherscanClient)
	store := service.NewAppStore()
	if !strings.EqualFold(os.Getenv("AUTO_IMPORT_WHALES_ON_START"), "false") {
		go func() {
			resp, err := store.ImportWhalesFromURL(ctx, "")
			if err != nil {
				log.Printf("[whales] auto import skipped: %v", err)
				return
			}
			log.Printf("[whales] auto imported %d rows from %s", resp.Imported, resp.Source)
		}()
	}
	priceService := service.NewPriceService()
	newsService := service.NewNewsService()
	notifyService := service.NewNotifyService()
	alertService := service.NewAlertService(store, etherscanClient, notifyService)
	alertService.StartScheduler(ctx)
	h := handler.NewHandler(etherscanClient, graphService, store, priceService, newsService, alertService)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "X-User-ID"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")
	{
		api.POST("/scan", h.ScanAddress)
		api.POST("/graph", h.GetGraph)
		api.POST("/balance", h.GetBalance)

		api.POST("/auth/google", h.LoginGoogle)
		api.POST("/auth/email", h.LoginGoogle)
		api.GET("/auth/google/start", h.StartGoogleOAuth)
		api.GET("/auth/google/callback", h.GoogleOAuthCallback)
		api.GET("/me", h.GetMe)
		api.GET("/whales", h.ListWhales)
		api.POST("/admin/whales/import-etherscan-csv", h.ImportEtherscanWhalesCSV)
		api.POST("/admin/whales/import-etherscan-url", h.ImportEtherscanWhalesURL)
		api.POST("/admin/jobs/run-watchlist-scan", h.RunWatchlistScan)
		api.GET("/addresses/:address", h.GetAddressDetail)
		api.GET("/addresses/:address/transactions", h.GetAddressTransactions)
		api.GET("/addresses/:address/graph", h.GetAddressGraph)
		api.GET("/addresses/:address/ai-summary", h.GetAddressAISummary)
		api.GET("/prices/eth/ohlc", h.GetETHPrices)
		api.GET("/news/eth", h.GetETHNews)
		api.GET("/watchlists", h.ListWatchlists)
		api.POST("/watchlists/confirm", h.UpsertWatchlistWithConfirmation)
		api.POST("/watchlists", h.UpsertWatchlist)
		api.PATCH("/watchlists/:id", h.UpsertWatchlist)
		api.DELETE("/watchlists/:id", h.DeleteWatchlist)
		api.GET("/alerts", h.ListAlerts)
		api.PATCH("/alerts/:id/read", h.MarkAlertRead)
		api.POST("/notification-preferences", h.UpdateNotificationPreferences)
		api.GET("/notifications/status", h.GetNotificationStatus)
		api.POST("/notifications/test", h.SendTestNotification)

		// Existing prototype APIs kept for compatibility.
		api.POST("/resolve-ens", h.ResolveENS)
		api.POST("/export", h.ExportCSV)
		api.POST("/gas-analytics", h.GetGasAnalytics)
		api.POST("/token-approvals", h.GetTokenApprovals)
		api.POST("/risk-score", h.GetRiskScore)
		api.POST("/contract-decode", h.DecodeContract)
	}

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
