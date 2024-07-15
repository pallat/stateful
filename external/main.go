package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var sysToken = map[string]string{}

type Auth struct {
	System string `json:"system"`
	Token  string `json:"token"`
}

func main() {
	r := gin.Default()

	r.GET("/token/:system", func(c *gin.Context) {
		system := c.Param("system")
		token := uuid.New().String()
		delete(sysToken, system)
		sysToken[system] = token

		c.JSON(http.StatusOK, gin.H{
			"token": token,
		})
	})

	r.POST("/token/verifies", func(c *gin.Context) {
		var auth Auth
		if err := c.ShouldBindJSON(&auth); err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		token, ok := sysToken[auth.System]
		if !ok {
			c.Status(http.StatusUnauthorized)
			return
		}

		if auth.Token != token {
			c.Status(http.StatusUnauthorized)
			return
		}

		c.Status(http.StatusOK)
	})

	srv := &http.Server{
		Addr:    ":8765",
		Handler: r,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go gracefully(ctx, srv)

	if err := srv.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}

	fmt.Println("\nbye")
}

func gracefully(ctx context.Context, srv *http.Server) {
	<-ctx.Done()
	{
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}
}
