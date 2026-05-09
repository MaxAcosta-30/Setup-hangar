// api/cmd/worker/main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"go.temporal.io/sdk/worker"

	"github.com/tu-usuario/hangar/api/internal/activities"
	"github.com/tu-usuario/hangar/api/internal/db"
	dockerfactory "github.com/tu-usuario/hangar/api/internal/docker"
	"github.com/tu-usuario/hangar/api/internal/rdb"
	temporalclient "github.com/tu-usuario/hangar/api/internal/temporal"
	"github.com/tu-usuario/hangar/api/internal/workflows"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file - usando variables de entorno")
	}

	// -- Dependencias --------------------------------------------------─
	pool, err := db.NewPool(context.Background())
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()
	log.Println("postgres conectado")

	dockerClient, err := dockerfactory.New()
	if err != nil {
		log.Fatalf("docker: %v", err)
	}
	defer dockerClient.Close()
	log.Println("docker engine conectado")

	redisClient, err := rdb.New()
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("redis conectado")

	tc, err := temporalclient.New()
	if err != nil {
		log.Fatalf("temporal: %v", err)
	}
	defer tc.Close()
	log.Println("temporal conectado")

	// -- Worker --------------------------------------------------------─
	w := worker.New(tc, temporalclient.TaskQueue, worker.Options{})

	w.RegisterWorkflow(workflows.DeployWorkflow)

	// Ahora las actividades tienen Redis para publicar logs en tiempo real
	deployActivities := activities.NewDeployActivities(pool, dockerClient, redisClient)
	w.RegisterActivity(deployActivities)

	log.Printf("worker iniciado - escuchando cola: %s", temporalclient.TaskQueue)

	// -- Graceful shutdown ----------------------------------------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	workerErr := make(chan error, 1)
	go func() {
		workerErr <- w.Run(worker.InterruptCh())
	}()

	select {
	case err := <-workerErr:
		if err != nil {
			log.Fatalf("worker error: %v", err)
		}
	case <-quit:
		log.Println("apagando worker...")
		w.Stop()
	}

	log.Println("worker apagado")
}
