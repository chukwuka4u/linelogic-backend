package services

import (
	"log"
	"net/http"
	"math/rand/v2"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type mockCreateQueue struct {
	QueueName string `json:"queueName" binding:"required"`
	// AttendedBy string `json:"attendedBy" biding:"required"`
}

type mockPassId struct {
	ID string `json:"id" binding:"required"`
}

type mockMemberAction struct {
	ID       string `json:"id" binding:"required"`
	MemberID string `json:"memberId" binding:"required"`
}

type mockMyQueues struct {
	IDList []string `json:"idList" binding:"required"`
}

// TODO: update the mockCreateQueue to contain time of attendance
// 	TODO: queue will be created using key "queue:[the id]"
func CreateQueue(c *gin.Context) {
	var req mockCreateQueue

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	var members = []redis.Z{
		{Score: 0, Member: "system:init"},
	}
	err := Rdb.ZAdd(Ctx, req.QueueName, members...).Err()
	if err != nil {
		log.Fatalf("Failed to create queue: %v", err)
	}
	c.JSON(201, gin.H{
		"created": req.QueueName,
	})
}

// TODO: passed the name as the id here, used as the queue key, refactor
func ReadQueue(c *gin.Context) {
	var req mockPassId

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})	
		return
	}

	result, err := Rdb.ZRange(Ctx, req.ID, 1, -1).Result()
	if err != nil {
		log.Fatalf("Failed to READ queue: %v", err)
	}
	
	c.JSON(200, gin.H{
		"result": result,
	})
}

func DeleteQueue(c *gin.Context) {
	var req mockPassId

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})	
		return
	}

	err := Rdb.ZDel(Ctx, req.ID).Err()
	if err != nil {
		log.Fatalf("Failed to READ queue: %v", err)
	}
	
	c.JSON(200, gin.H{
		"deleted": req.ID,
	})
}

func RemoveMember(c *gin.Context) {
	var req mockMemberAction

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := Rdb.ZRem(Ctx, req.ID, req.MemberID).Err()
	if err != nil {
		log.Fatalf("Failed to READ queue: %v", err)
	}
	
	c.JSON(200, gin.H{
		"removed member": req.MemberID,
		"from queue": req.ID,
	})
}

func JoinQueue(c *gin.Context) {
	var req mockMemberAction

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	randomNumber := rand.Intn(50) + 1
	var members = []redis.Z{
		{Score: randomNumber, Member: req.MemberID},
	}
	err := Rdb.ZAdd(Ctx, req.ID, members...).Err()
	if err != nil {
		log.Fatalf("Failed to create queue: %v", err)
	}
	c.JSON(201, gin.H{
		"joined member": req.MemberID,
		"to queue": req.ID,
	})
}

func LeaveQueue() {
	// var req mockMemberAction
}

// store queue ids in client, use MyQueues to fetch some fields from
// some queues
func MyQueues() {
	// var req mockMyQueues
}
