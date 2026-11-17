package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	loginURL       = getEnv("LOGIN_URL", "http://localhost:8080/api/auth/login")
	numRequests    = 15
	leakIndicators = []string{
		"traceback",
		"stack trace",
		"panic:",
		".go:",
		"goroutine",
		"syntax error",
		"database/sql",
		"sqlite",
		"pq:",
		"internal server error",
		"exception",
	}
)

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getPayload() []byte {
	data, _ := json.Marshal(map[string]string{
		"username": "test_invalid",
		"password": "invalid_pass",
		"user":     "test_invalid",
		"pass":     "invalid_pass",
	})
	return data
}

func assertNoInternalLeaks(t *testing.T, reqID int, body string) {
	t.Helper()
	lower := strings.ToLower(body)
	for _, pattern := range leakIndicators {
		if strings.Contains(lower, pattern) {
			t.Fatalf("[LEAK] Petición #%d expuso detalles internos (%q): %s", reqID, pattern, body)
		}
	}
}

func postLogin(client *http.Client, payload []byte) (int, string, time.Duration, error) {
	start := time.Now()
	resp, err := client.Post(loginURL, "application/json", bytes.NewReader(payload))
	elapsed := time.Since(start)
	if err != nil {
		return 0, "", elapsed, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", elapsed, err
	}
	return resp.StatusCode, string(body), elapsed, nil
}

func calcStats(durations []time.Duration) (min, max, avg time.Duration, stddevMs float64) {
	if len(durations) == 0 {
		return
	}
	min, max = durations[0], durations[0]
	var sum time.Duration
	for _, d := range durations {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
		sum += d
	}
	avg = sum / time.Duration(len(durations))

	avgMs := float64(avg.Microseconds()) / 1000.0
	var sumSq float64
	for _, d := range durations {
		dMs := float64(d.Microseconds()) / 1000.0
		diff := dMs - avgMs
		sumSq += diff * diff
	}
	if len(durations) > 1 {
		stddevMs = math.Sqrt(sumSq / float64(len(durations)-1))
	}
	return
}

func TestAuthRateLimitingAndTimingDefense(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	payload := getPayload()

	durations := make([]time.Duration, 0, numRequests)
	statuses := make([]int, 0, numRequests)
	has429 := false

	t.Logf("Iniciando secuencia de %d peticiones a %s", numRequests, loginURL)

	for i := 1; i <= numRequests; i++ {
		status, body, elapsed, err := postLogin(client, payload)
		if err != nil {
			t.Fatalf("Error de conexión en petición #%d hacia %s: %v", i, loginURL, err)
		}

		durations = append(durations, elapsed)
		statuses = append(statuses, status)
		if status == http.StatusTooManyRequests {
			has429 = true
		}

		assertNoInternalLeaks(t, i, body)
		t.Logf("Req %02d -> HTTP %d | %.2f ms", i, status, float64(elapsed.Microseconds())/1000.0)
	}

	minD, maxD, avgD, stddevMs := calcStats(durations)
	t.Logf("Resumen Latencia: Min=%.2fms | Max=%.2fms | Avg=%.2fms | StdDev=%.2fms",
		float64(minD.Microseconds())/1000.0,
		float64(maxD.Microseconds())/1000.0,
		float64(avgD.Microseconds())/1000.0,
		stddevMs,
	)

	if !has429 {
		t.Fatalf("Fallo de seguridad: Tras %d intentos fallidos nunca devolvió HTTP 429. Códigos: %v",
			numRequests, statuses)
	}
}

func TestConcurrentBurstDegradation(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	payload := getPayload()

	type result struct {
		id      int
		status  int
		body    string
		elapsed time.Duration
		err     error
	}

	results := make([]result, numRequests)
	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			status, body, elapsed, err := postLogin(client, payload)
			results[idx] = result{id: idx + 1, status: status, body: body, elapsed: elapsed, err: err}
		}(i)
	}
	wg.Wait()

	durations := make([]time.Duration, 0, numRequests)
	statuses := make([]int, 0, numRequests)
	has429 := false

	for _, r := range results {
		if r.err != nil {
			t.Fatalf("Error en ráfaga concurrente req #%d: %v", r.id, r.err)
		}
		statuses = append(statuses, r.status)
		durations = append(durations, r.elapsed)
		if r.status == http.StatusTooManyRequests {
			has429 = true
		}
		assertNoInternalLeaks(t, r.id, r.body)
	}

	_, _, avgD, _ := calcStats(durations)
	t.Logf("Ráfaga concurrente completada: Códigos=%v | Avg=%.2fms",
		statuses, float64(avgD.Microseconds())/1000.0)

	if !has429 {
		t.Fatalf("Fallo de seguridad: Ráfaga concurrente no activó HTTP 429. Códigos: %v", statuses)
	}
}
