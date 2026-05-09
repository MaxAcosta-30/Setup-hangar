// api/cmd/server/main.go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/MaxAcosta-30/hangar/api/internal/db"
	"github.com/MaxAcosta-30/hangar/api/internal/handler"
	"github.com/MaxAcosta-30/hangar/api/internal/middleware"
	"github.com/MaxAcosta-30/hangar/api/internal/rdb"
	temporalclient "github.com/MaxAcosta-30/hangar/api/internal/temporal"
	dockerfactory "github.com/MaxAcosta-30/hangar/api/internal/docker"
)

func main() {
	if err := godotenv.Load(); err != nil {
		// No fatal en producción
	}

	log, _ := zap.NewDevelopment()
	defer log.Sync()

	// -- Dependencias --------------------------------------------------─
	pool, err := db.NewPool(context.Background())
	if err != nil {
		log.Fatal("postgres", zap.Error(err))
	}
	defer pool.Close()
	log.Info("postgres conectado")

	redisClient, err := rdb.New()
	if err != nil {
		log.Fatal("redis", zap.Error(err))
	}
	defer redisClient.Close()
	log.Info("redis conectado")

	tc, err := temporalclient.New()
	if err != nil {
		log.Fatal("temporal", zap.Error(err))
	}
	defer tc.Close()
	log.Info("temporal conectado")

	dockerClient, err := dockerfactory.New()
	if err != nil {
		log.Fatal("docker", zap.Error(err))
	}
	defer dockerClient.Close()
	log.Info("docker engine conectado")

	// -- Repositorios --------------------------------------------------─
	appRepo    := db.NewAppRepository(pool)
	deployRepo := db.NewDeploymentRepository(pool)

	// -- Handlers ------------------------------------------------------─
	healthHandler := handler.NewHealthHandler(pool)
	appHandler    := handler.NewAppHandler(appRepo, deployRepo, tc, dockerClient, redisClient, log)
	wsHandler     := handler.NewWSHandler(deployRepo, redisClient, log)

	// -- Servidor HTTP --------------------------------------------------
	app := fiber.New(fiber.Config{
		AppName:      "Hangar API",
		ErrorHandler: middleware.ErrorHandler(log),
	})

	app.Use(middleware.Logger(log))

	// Rutas REST
	app.Get("/health", healthHandler.Check)
	api := app.Group("/api/v1")
	appHandler.Register(api)

	// Ruta WebSocket - debe declararse ANTES de usar websocket.New
	// El middleware IsWebSocketUpgrade verifica que sea una conexión WS válida
	api.Get("/deployments/:id/stream",
		func(c *fiber.Ctx) error {
			if websocket.IsWebSocketUpgrade(c) {
				return c.Next()
			}
			return fiber.ErrUpgradeRequired
		},
		websocket.New(wsHandler.StreamLogs),
	)

	// -- Arranque graceful ----------------------------------------------
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("servidor iniciando", zap.String("port", port))
		if err := app.Listen(":" + port); err != nil {
			log.Error("error en servidor", zap.Error(err))
		}
	}()

	<-quit
	log.Info("apagando servidor...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("error en shutdown", zap.Error(err))
	}

	log.Info("servidor apagado")
}
