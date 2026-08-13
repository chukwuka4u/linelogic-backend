package services

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chukwuka4u/linelogic-backend/token"
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
	payload, exists := c.Get("authorization_payload")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "payload lost, retry request"})
		return
	}

	var ch = make(chan string)
	go func(pool *pgxpool.Pool) {
		id, err := CreateQueueDB(req.QueueName, req.Department, payload.(*token.Payload).UserID, pool)
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

type mockBrowseQueue struct {
	ID         string    `json:"id" binding:"required"`
	QueueName  string    `json:"queueName" binding:"required"`
	Department string    `json:"department" binding:"required"`
	WaitTime   time.Time `json:"waitTime"`
	NoWaiting  int       `json:"noWaiting"`
}

func BrowseQueue(c *gin.Context) {
	ls := make(chan []mockBrowseQueue)
	go func(pool *pgxpool.Pool) {
		// fetch queuename and department from DB for each queue ID in req.IDList
		// waiting time will be calculated based on the number of members in the queue and the average service time
		// For simplicity, let's assume each member takes 5 minutes on average to be served
		// NoWaiting will be gotten from zset counting the number of members in a set.
		queues, err := BrowseQueueDB(pool)
		if err != nil {
			log.Printf("BrowseQueueDB error: %v", err)
			ls <- nil
		} else {
			ls <- queues
		}
	}(DB)
	queueListPartial := <-ls
	if queueListPartial == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to browse queues in DB"})
		return
	}

	for i := range queueListPartial {
		queueKey := fmt.Sprintf("queue:%s", queueListPartial[i].ID)
		count, err := Rdb.ZCard(Ctx, queueKey).Result()
		if err != nil {
			log.Printf("Failed to get number of waiting members for queue %s: %v", queueKey, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get number of waiting members"})
			return
		}
		queueListPartial[i].NoWaiting = int(count) - 1 // Subtracting 1 to exclude the "system:init" member

		// Assuming each member takes 5 minutes on average to be served
		waitTime := time.Duration(count*5) * time.Minute
		queueListPartial[i].WaitTime = time.Now().Add(waitTime)
	}

	c.JSON(http.StatusOK, gin.H{"queues": queueListPartial})
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

	payload, exists := c.Get("authorization_payload")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "payload lost, retry request"})
		return
	}

	if err := JoinUpdateDB(req.ID, payload.(*token.Payload).UserID); err != nil {
		log.Printf("JoinUpdateDB error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record queue membership"})
		return
	}

	members := []redis.Z{{Score: float64(time.Now().Unix()), Member: req.MemberID}}
	queueKey := fmt.Sprintf("queue:%s", req.ID)
	err := Rdb.ZAdd(Ctx, queueKey, members...).Err()
	if err != nil {
		log.Printf("Failed to add member to queue %s: %v", queueKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join queue"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"joined member": payload.(*token.Payload).UserID,
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

func Migration(c *gin.Context) {
	err := MigrateAction()
	if err != nil {
		log.Printf("migration failed %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "migration failed"})
		return
	}
}
