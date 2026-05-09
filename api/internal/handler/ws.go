// api/internal/handler/ws.go
package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/MaxAcosta-30/hangar/api/internal/db"
	"github.com/MaxAcosta-30/hangar/api/internal/rdb"
)

// WSHandler maneja el streaming de logs via WebSocket.
type WSHandler struct {
	deploys *db.DeploymentRepository
	redis   *redis.Client
	log     *zap.Logger
}

// NewWSHandler crea un nuevo handler de WebSocket.
func NewWSHandler(
	deploys *db.DeploymentRepository,
	redis *redis.Client,
	log *zap.Logger,
) *WSHandler {
	return &WSHandler{
		deploys: deploys,
		redis:   redis,
		log:     log,
	}
}

// LogMessage es el formato JSON que enviamos por el WebSocket.
type LogMessage struct {
	Type    string `json:"type"`    // "log" | "status" | "done" | "error"
	Message string `json:"message"`
	Time    string `json:"time"`
}

// StreamLogs es el handler WebSocket para:
//   ws://localhost:3000/api/v1/deployments/:id/stream
//
// Flujo:
//  1. Envía todos los logs históricos (para clientes que conectan tarde)
//  2. Se suscribe al canal Redis del deployment
//  3. Reenvía cada log nuevo al browser en tiempo real
//  4. Cierra la conexión cuando el deployment termina (success o failed)
func (h *WSHandler) StreamLogs(c *websocket.Conn) {
	deploymentID := c.Params("id")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	h.log.Info("websocket conectado", zap.String("deployment_id", deploymentID))

	// -- 1. Envía logs históricos ----------------------------------------
	// Útil cuando el cliente conecta después de que el deploy empezó
	historicalLogs, err := h.deploys.GetLogs(ctx, deploymentID)
	if err != nil {
		h.sendMessage(c, LogMessage{
			Type:    "error",
			Message: "deployment no encontrado",
			Time:    now(),
		})
		return
	}

	for _, l := range historicalLogs {
		h.sendMessage(c, LogMessage{
			Type:    "log",
			Message: l.Message,
			Time:    l.LoggedAt.Format(time.RFC3339),
		})
	}

	// Si el deployment ya terminó, cierra inmediatamente después del historial
	deployment, _ := h.deploys.GetByID(ctx, deploymentID)
	if deployment != nil && isTerminal(string(deployment.Status)) {
		h.sendMessage(c, LogMessage{
			Type:    "done",
			Message: string(deployment.Status),
			Time:    now(),
		})
		return
	}

	// -- 2. Suscribe al canal Redis para logs en vivo --------------------
	channel := rdb.LogChannel(deploymentID)
	pubsub := h.redis.Subscribe(ctx, channel)
	defer pubsub.Close()

	h.log.Info("suscrito a redis",
		zap.String("channel", channel),
		zap.String("deployment_id", deploymentID),
	)

	// -- 3. Reenvía mensajes en tiempo real ----------------------------─
	msgChan := pubsub.Channel()

	// Ticker para verificar si el deployment terminó
	// (por si el mensaje de "done" se perdió)
	statusTicker := time.NewTicker(3 * time.Second)
	defer statusTicker.Stop()

	for {
		select {

		// Nuevo log publicado por una actividad de Temporal
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			h.sendMessage(c, LogMessage{
				Type:    "log",
				Message: msg.Payload,
				Time:    now(),
			})

		// Verifica el status cada 3 segundos para detectar fin del deploy
		case <-statusTicker.C:
			d, err := h.deploys.GetByID(ctx, deploymentID)
			if err != nil {
				continue
			}
			if isTerminal(string(d.Status)) {
				h.sendMessage(c, LogMessage{
					Type:    "done",
					Message: string(d.Status),
					Time:    now(),
				})
				return
			}

		// El cliente cerró la conexión
		case <-ctx.Done():
			return
		}
	}
}

// -- Helpers --------------------------------------------------------------─

func (h *WSHandler) sendMessage(c *websocket.Conn, msg LogMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
		h.log.Debug("error enviando mensaje ws", zap.Error(err))
	}
}

func isTerminal(status string) bool {
	return status == "success" || status == "failed" || status == "canceled"
}

func now() string {
	return time.Now().Format(time.RFC3339)
}
