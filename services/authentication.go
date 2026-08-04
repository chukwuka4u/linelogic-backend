package services

import (
	"net/http"
	"os"

	"github.com/chukwuka4u/linelogic-backend/token"
	"github.com/gin-gonic/gin"
)

type mockLogin struct {
	Username string `json:"username" binding:"required"`
	UserId   string `json:"userId" binding:"required"`
}

type loginResponse struct {
	APIAccessToken string `json:"apiAccessToken" binding:"required"`
}
type ValidUserResult struct {
	ID  string
	Err error
}

// TODO: useless for now, we have to first build the sync feature first!
func Login(c *gin.Context) {
	var req mockLogin
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}

	ch := make(chan ValidUserResult)
	go func() {
		id, err := ValidUser(req.UserId, req.Username)
		if err != nil {
			ch <- ValidUserResult{ID: "", Err: err}
		} else {
			ch <- ValidUserResult{ID: id, Err: nil}
		}
	}()

	check := <-ch
	if check.Err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": check.Err.Error(),
		})
		return
	}
	//create new token
	symKey := os.Getenv("TOKEN_SYMMETRIC_KEY")
	maker, err := token.NewPasetoMaker(symKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	token, err := maker.CreateToken(check.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	rsp := &loginResponse{
		APIAccessToken: token,
	}
	c.JSON(http.StatusAccepted, rsp)

}

type mockUser struct {
	Username string `json:"username" binding:"required"`
	UserID   string `json:"userId" binding:"required"`
	Phone    string `json:"phone" binding:"required" validate:"max=11"`
}

// login immediately after sync for signups
func SyncUser(c *gin.Context) {
	var req mockUser
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}

	ch := make(chan ValidUserResult)
	go func() {
		id, err := CreateUser(req.Username, req.UserID, req.Phone)
		if err != nil {
			ch <- ValidUserResult{ID: "", Err: err}
		} else {
			ch <- ValidUserResult{ID: id, Err: nil}
		}
	}()

	check := <-ch
	if check.Err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": check.Err.Error(),
		})
		return
	}
	//create new token
	symKey := os.Getenv("TOKEN_SYMMETRIC_KEY")
	maker, err := token.NewPasetoMaker(symKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	token, err := maker.CreateToken(check.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	rsp := &loginResponse{
		APIAccessToken: token,
	}
	c.JSON(http.StatusAccepted, rsp)
}
