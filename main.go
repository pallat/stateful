package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type Request struct {
	System string `json:"system"`
	Token  string `json:"token"`
}

type Token struct {
	Token string `json:"token"`
}

type tokenCache struct {
	mux   sync.Mutex
	token string
}

var tokenInstance = tokenCache{}

func main() {
	r := gin.Default()

	r.GET("/api", func(c *gin.Context) {
		tokenInstance.mux.Lock()
		t := tokenInstance.token
		tokenInstance.mux.Unlock()

		req := Request{
			System: "arise",
			Token:  t,
		}

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(&req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		resp, err := http.Post("http://localhost:8765/token/verifies", "application/json", &buf)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		if resp.StatusCode == http.StatusOK {
			c.Status(http.StatusOK)
			return
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp, err := http.Get("http://localhost:8765/token/arise")
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			if resp.StatusCode != http.StatusOK {
				c.Status(resp.StatusCode)
				return
			}

			defer resp.Body.Close()

			var t Token
			if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": err.Error(),
				})
				return
			}

			tokenInstance.mux.Lock()
			tokenInstance.token = t.Token
			tokenInstance.mux.Unlock()

			req := Request{
				System: "arise",
				Token:  tokenInstance.token,
			}

			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(&req); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"message": err.Error(),
				})
				return
			}

			resp, err = http.Post("http://localhost:8765/token/verifies", "application/json", &buf)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": err.Error(),
				})
				return
			}

			if resp.StatusCode != http.StatusOK {
				c.Status(resp.StatusCode)
				return
			}

			c.Status(http.StatusOK)
			return
		}

		c.Status(resp.StatusCode)
	})

	srv := &http.Server{
		Addr:    ":" + os.Getenv("PORT"),
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
