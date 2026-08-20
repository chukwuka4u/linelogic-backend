package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chukwuka4u/linelogic-backend/services"
	"github.com/chukwuka4u/linelogic-backend/token"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables", err)
	}
}

func main() {
	services.RedisConnect()
	services.PostGresConnect()

	hub := services.NewHub()

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "up",
		})
	})

	router.GET("/redis-test", func(c *gin.Context) {

		val, err := services.Rdb.Get(services.Ctx, "foo").Result()
		if err != nil {
			log.Fatal("Error getting Redis key:", err)
		}
		c.JSON(200, gin.H{
			"foo": val,
		})
	})
	router.GET("/migrate", services.Migration)
	router.POST("/sync-user", services.SyncUser)
	router.POST("/login", services.Login)
	router.GET("/stream", func(c *gin.Context) {
		services.StreamTicker(c, hub)
	})

	sym_key := os.Getenv("TOKEN_SYMMETRIC_KEY")
	tokenMaker, err := token.NewPasetoMaker(sym_key)
	if err != nil {
		log.Fatal("failed to create token maker:", err)
	}

	pasetoMaker, ok := tokenMaker.(*token.PasetoMaker)
	if !ok {
		log.Fatal("token maker is not a PasetoMaker")
	}
	authRouter := router.Group("/").Use(authMiddleware(pasetoMaker))

	authRouter.POST("/create-queue", services.CreateQueue)
	authRouter.POST("/queues", services.BrowseQueue)
	authRouter.POST("/read-queue", services.ReadQueue)
	authRouter.POST("/delete-queue", services.DeleteQueue)
	authRouter.POST("/remove-member", services.RemoveMember)
	authRouter.POST("/join-queue", func(c *gin.Context) {
		services.JoinQueue(c, hub)
	})
	authRouter.POST("/leave-queue", services.LeaveQueue)

	myapi := os.Getenv("MY_API")
	startSelfPing(myapi+"/health", 10*time.Minute)

	fmt.Println("Server running on port 8080...")
	router.Run()
}
