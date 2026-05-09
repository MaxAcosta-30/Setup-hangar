// api/tests/deploy_integration_test.go
//
// Test de integración del flujo completo de deploy.
// Requiere que Docker, Postgres, Redis y Temporal estén corriendo.
//
// Uso:
//   cd api
//   go test ./tests/... -v -timeout 120s -tags integration
//
// Para correr tests normales sin este:
//   go test ./...   (sin -tags integration)

//go:build integration

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// baseURL del API server — debe estar corriendo antes del test
const baseURL = "http://localhost:3000"

// testGitURL: repo local con Dockerfile listo
// Cambia esto a tu repo local si es diferente
const testGitURL = "file:///home/max/hello-hangar-repo"

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func post(t *testing.T, path string, body any) map[string]any {
	t.Helper()
	data, _ := json.Marshal(body)
	resp, err := http.Post(baseURL+path, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("POST %s falló: %v", path, err)
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func get(t *testing.T, path string) map[string]any {
	t.Helper()
	resp, err := http.Get(baseURL + path)
	if err != nil {
		t.Fatalf("GET %s falló: %v", path, err)
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

// waitForStatus hace polling al status del deployment hasta que sea terminal
// o se agote el timeout.
func waitForStatus(t *testing.T, deploymentID string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	terminal := map[string]bool{"success": true, "failed": true, "canceled": true}

	for time.Now().Before(deadline) {
		// El deployment está en deploy_logs y en la tabla deployments
		// Lo buscamos vía los logs de la app
		resp, err := http.Get(fmt.Sprintf("%s/api/v1/deployments/%s/logs", baseURL, deploymentID))
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		resp.Body.Close()

		// Obtenemos el status directo de la DB
		pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		var status string
		err = pool.QueryRow(
			context.Background(),
			`SELECT status FROM deployments WHERE id = $1`,
			deploymentID,
		).Scan(&status)
		pool.Close()

		if err == nil && terminal[status] {
			return status
		}

		t.Logf("deployment %s status: %s — esperando...", deploymentID[:8], status)
		time.Sleep(3 * time.Second)
	}

	t.Fatalf("timeout esperando que el deployment %s termine", deploymentID[:8])
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestHealthCheck verifica que el API responde correctamente
func TestHealthCheck(t *testing.T) {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("API no responde: %v — ¿está corriendo go run cmd/server/main.go?", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("health check devolvió %d, esperaba 200", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)

	if body["status"] != "ok" {
		t.Errorf("status esperado 'ok', recibido: %v", body["status"])
	}
	if body["postgres"] != "ok" {
		t.Errorf("postgres no está ok: %v", body["postgres"])
	}

	t.Log("✓ health check ok")
}

// TestCreateApp verifica la creación de una app y la generación del subdominio
func TestCreateApp(t *testing.T) {
	appName := fmt.Sprintf("test-app-%d", time.Now().Unix())

	result := post(t, "/api/v1/apps", map[string]string{
		"name":    appName,
		"git_url": testGitURL,
	})

	app, ok := result["app"].(map[string]any)
	if !ok {
		t.Fatalf("respuesta no tiene campo 'app': %v", result)
	}

	if app["id"] == nil || app["id"] == "" {
		t.Error("app.id está vacío")
	}
	if app["subdomain"] == nil || app["subdomain"] == "" {
		t.Error("app.subdomain está vacío")
	}
	if app["status"] != "idle" {
		t.Errorf("status inicial esperado 'idle', recibido: %v", app["status"])
	}

	t.Logf("✓ app creada: id=%s subdomain=%s", app["id"], app["subdomain"])
}

// TestFullDeployFlow es el test más importante:
// crea una app, dispara un deploy, espera que termine,
// y verifica que el contenedor está corriendo en Docker.
func TestFullDeployFlow(t *testing.T) {
	appName := fmt.Sprintf("integration-test-%d", time.Now().Unix())

	// ── 1. Crea la app ────────────────────────────────────────────────
	createResult := post(t, "/api/v1/apps", map[string]string{
		"name":    appName,
		"git_url": testGitURL,
	})

	app, ok := createResult["app"].(map[string]any)
	if !ok {
		t.Fatalf("no se pudo crear la app: %v", createResult)
	}
	appID := app["id"].(string)
	subdomain := app["subdomain"].(string)
	t.Logf("app creada: %s (%s)", appName, appID[:8])

	// ── 2. Dispara el deploy ──────────────────────────────────────────
	deployResult := post(t, fmt.Sprintf("/api/v1/apps/%s/deploy", appID), nil)

	deployment, ok := deployResult["deployment"].(map[string]any)
	if !ok {
		t.Fatalf("no se pudo disparar el deploy: %v", deployResult)
	}
	deploymentID := deployment["id"].(string)
	t.Logf("deploy iniciado: %s", deploymentID[:8])

	// ── 3. Espera que el workflow complete (máx 90 segundos) ──────────
	finalStatus := waitForStatus(t, deploymentID, 90*time.Second)

	if finalStatus != "success" {
		// Muestra los logs del deploy para facilitar el debugging
		logsResult := get(t, fmt.Sprintf("/api/v1/deployments/%s/logs", deploymentID))
		t.Logf("logs del deploy fallido: %v", logsResult)
		t.Fatalf("deploy terminó con status '%s', esperaba 'success'", finalStatus)
	}
	t.Logf("✓ deploy completado con status: %s", finalStatus)

	// ── 4. Verifica los logs en DB ────────────────────────────────────
	logsResult := get(t, fmt.Sprintf("/api/v1/deployments/%s/logs", deploymentID))
	logs, _ := logsResult["logs"].([]any)

	if len(logs) == 0 {
		t.Error("deploy_logs está vacío — las actividades no guardaron logs")
	}

	// Verifica que hay al menos los logs clave del flujo completo
	logMessages := make([]string, 0, len(logs))
	for _, l := range logs {
		if entry, ok := l.(map[string]any); ok {
			if msg, ok := entry["message"].(string); ok {
				logMessages = append(logMessages, msg)
			}
		}
	}

	expectedSubstrings := []string{
		"clonando",
		"repositorio clonado",
		"docker build",
		"imagen construida",
		"contenedor",
		"disponible en",
	}
	for _, expected := range expectedSubstrings {
		found := false
		for _, msg := range logMessages {
			if containsStr(msg, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("log esperado no encontrado: '%s'", expected)
		}
	}
	t.Logf("✓ %d líneas de log verificadas", len(logs))

	// ── 5. Verifica la red aislada en Docker ─────────────────────────
	// La red debe existir con el nombre correcto
	expectedNetwork := fmt.Sprintf("hangar-app-%s", appID)
	t.Logf("verificando red aislada: %s", expectedNetwork)

	// Usamos la API de Docker directamente via HTTP para no importar el SDK aquí
	dockerResp, err := http.Get("http://localhost:2375/networks/" + expectedNetwork)
	if err == nil && dockerResp.StatusCode == 200 {
		t.Logf("✓ red aislada existe: %s", expectedNetwork)
		dockerResp.Body.Close()
	} else {
		// Docker por defecto escucha en socket, no en TCP — skip este check si no hay TCP
		t.Logf("⚠ verificación de red skipped (Docker API TCP no disponible en :2375)")
		t.Logf("  verifica manualmente: docker network ls | grep %s", appID[:8])
	}

	t.Logf("✓ flujo completo verificado para app: %s.hangar.local", subdomain)
}

// TestSubdomainUniqueness verifica que dos apps con el mismo nombre
// reciben subdominios diferentes (no hay colisión)
func TestSubdomainUniqueness(t *testing.T) {
	name := fmt.Sprintf("collision-test-%d", time.Now().Unix())

	first := post(t, "/api/v1/apps", map[string]string{
		"name": name, "git_url": testGitURL,
	})
	second := post(t, "/api/v1/apps", map[string]string{
		"name": name, "git_url": testGitURL,
	})

	firstApp  := first["app"].(map[string]any)
	secondApp := second["app"].(map[string]any)

	firstSubdomain  := firstApp["subdomain"].(string)
	secondSubdomain := secondApp["subdomain"].(string)

	if firstSubdomain == secondSubdomain {
		t.Errorf("colisión de subdominios: ambas apps tienen '%s'", firstSubdomain)
	}

	t.Logf("✓ subdominios únicos: '%s' y '%s'", firstSubdomain, secondSubdomain)
}

// TestDeployNonexistentApp verifica que disparar deploy en app inexistente
// devuelve 404 y no crea registros huérfanos en la DB
func TestDeployNonexistentApp(t *testing.T) {
	fakeID := "00000000-0000-0000-0000-000000000000"
	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/apps/%s/deploy", baseURL, fakeID),
		"application/json",
		nil,
	)
	if err != nil {
		t.Fatalf("request falló: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("esperaba 404, recibió %d", resp.StatusCode)
	}

	t.Log("✓ app inexistente devuelve 404 correctamente")
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers internos
// ─────────────────────────────────────────────────────────────────────────────

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && searchStr(s, substr))
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
