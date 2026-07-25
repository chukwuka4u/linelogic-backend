package services

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type mockCreateQueue struct {
	QueueName  string `json:"queueName" binding:"required"`
	Department string `json:"department" binding:"required"`
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

func CreateQueue(c *gin.Context) {
	var req mockCreateQueue

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var ch = make(chan string)
	go func(pool *pgxpool.Pool) {
		id, err := CreateQueueDB(req.QueueName, req.Department, pool)
		if err != nil {
			log.Printf("CreateQueueDB error: %v", err)
			ch <- ""
		} else {
			ch <- id
		}
	}(DB)
	id := <-ch
	if id == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create queue in DB"})
		return
	}

	queueKey := fmt.Sprintf("queue:%s", id)
	members := []redis.Z{{Score: 0, Member: "system:init"}}
	err := Rdb.ZAdd(Ctx, queueKey, members...).Err()
	if err != nil {
		log.Printf("Failed to create queue in Redis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize queue"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"created": queueKey,
		"id":      id,
	})
}

func ReadQueue(c *gin.Context) {
	var req mockPassId
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	queueKey := fmt.Sprintf("queue:%s", req.ID)
	result, err := Rdb.ZRangeWithScores(Ctx, queueKey, 1, -1).Result()
	if err != nil {
		log.Printf("Failed to read queue %s: %v", queueKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read queue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": result})
}

func DeleteQueue(c *gin.Context) {
	var req mockPassId
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	queueKey := fmt.Sprintf("queue:%s", req.ID)
	err := Rdb.Del(Ctx, queueKey).Err()
	if err != nil {
		log.Printf("Failed to delete queue %s: %v", queueKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete queue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": queueKey})
}

func RemoveMember(c *gin.Context) {
	var req mockMemberAction
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	queueKey := fmt.Sprintf("queue:%s", req.ID)
	err := Rdb.ZRem(Ctx, queueKey, req.MemberID).Err()
	if err != nil {
		log.Printf("Failed to remove member %s from queue %s: %v", req.MemberID, queueKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"removed member": req.MemberID,
		"from queue":     queueKey,
	})
}

func JoinQueue(c *gin.Context) {
	var req mockMemberAction
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	queueID, err := strconv.Atoi(req.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "queue id must be a valid number"})
		return
	}

	if err = JoinUpdateDB(queueID, req.MemberID); err != nil {
		log.Printf("JoinUpdateDB error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record queue membership"})
		return
	}

	randomNumber := rand.IntN(50) + 1
	members := []redis.Z{{Score: float64(randomNumber), Member: req.MemberID}}
	queueKey := fmt.Sprintf("queue:%s", req.ID)
	err = Rdb.ZAdd(Ctx, queueKey, members...).Err()
	if err != nil {
		log.Printf("Failed to add member to queue %s: %v", queueKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join queue"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"joined member": req.MemberID,
		"to queue":      queueKey,
	})
}

func LeaveQueue(c *gin.Context) {
	var req mockMemberAction
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	queueKey := fmt.Sprintf("queue:%s", req.ID)
	err := Rdb.ZRem(Ctx, queueKey, req.MemberID).Err()
	if err != nil {
		log.Printf("Failed to leave queue %s: %v", queueKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to leave queue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"left member": req.MemberID,
		"from queue":  queueKey,
	})
}
