package main

import (
	"eth-sweeper/handler"
	"eth-sweeper/service"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	etherscanClient := service.NewEtherscanClient()
	graphService := service.NewGraphService(etherscanClient)
	h := handler.NewHandler(etherscanClient, graphService)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")
	{
		api.POST("/scan", h.ScanAddress)
		api.POST("/graph", h.GetGraph)
		api.POST("/balance", h.GetBalance)

		// TODO: 待實作的 API
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
