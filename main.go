package main

import (
	"fmt"
	"log"

	"github.com/chukwuka4u/linelogic-backend/services"
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

	router.POST("/create-queue", services.CreateQueue)
	router.POST("/read-queue", services.ReadQueue)
	router.POST("/delete-queue", services.DeleteQueue)
	router.POST("/remove-member", services.RemoveMember)
	router.POST("/join-queue", services.JoinQueue)
	router.POST("/leave-queue", services.LeaveQueue)

	fmt.Println("Server running on port 8080...")
	router.Run()
}
