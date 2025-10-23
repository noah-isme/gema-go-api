package performance_test

import (
	"bufio"
	"context"
	"errors"
	"math"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/handler"
	"github.com/noah-isme/gema-go-api/internal/middleware"
	"github.com/noah-isme/gema-go-api/internal/service"
)

func TestRealtimeChatWebsocketP95Under250ms(t *testing.T) {
	app := fiber.New()
	app.Use(middleware.CorrelationID())

	chatService := &stubChatService{}
	chatHandler := handler.NewChatHandler(chatService, validator.New(), zerolog.Nop())

	chatGroup := app.Group("/api/v2/chat", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(42))
		c.Locals("user_role", "student")
		return c.Next()
	})
	chatHandler.Register(chatGroup)

	baseURL, shutdown := startFiberServer(t, app)
	defer shutdown()

	url := "ws" + strings.TrimPrefix(baseURL, "http") + "/api/v2/chat/ws?room_id=room-1"
	clients := 500
	durations := make([]time.Duration, 0, clients)

	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}

	for i := 0; i < clients; i++ {
		start := time.Now()
		conn, resp, err := dialer.Dial(url, http.Header{"X-Correlation-ID": {"perf-" + strconv.Itoa(i)}})
		if err != nil {
			t.Fatalf("websocket dial failed: %v", err)
		}
		if resp != nil {
			_ = resp.Body.Close()
		}

		_, _, _ = conn.ReadMessage()
		_ = conn.Close()

		durations = append(durations, time.Since(start))
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p95 := percentile(durations, 0.95)

	if p95 > 250*time.Millisecond {
		t.Fatalf("expected websocket P95 <= 250ms, got %s", p95)
	}
}

func TestRealtimeNotificationsSSEP95Under300ms(t *testing.T) {
	app := fiber.New()
	app.Use(middleware.CorrelationID())

	notifications := handler.NewNotificationHandler(&stubNotificationService{}, zerolog.Nop(), 30*time.Second)

	notificationsGroup := app.Group("/api/v2/notifications", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(7))
		return c.Next()
	})
	notifications.Register(notificationsGroup)

	baseURL, shutdown := startFiberServer(t, app)
	defer shutdown()

	client := &http.Client{Timeout: 5 * time.Second}
	clients := 200
	durations := make([]time.Duration, 0, clients)

	for i := 0; i < clients; i++ {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v2/notifications/stream", nil)
		if err != nil {
			t.Fatalf("build request failed: %v", err)
		}

		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("sse request failed: %v", err)
		}

		reader := bufio.NewReader(resp.Body)
		deadline := time.Now().Add(2 * time.Second)

		for {
			if time.Now().After(deadline) {
				t.Fatalf("sse response timed out for client %d", i)
			}
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("failed to read sse line: %v", err)
			}
			if strings.HasPrefix(line, "data:") {
				durations = append(durations, time.Since(start))
				break
			}
		}

		resp.Body.Close()
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p95 := percentile(durations, 0.95)

	if p95 > 300*time.Millisecond {
		t.Fatalf("expected SSE P95 <= 300ms, got %s", p95)
	}
}

func percentile(values []time.Duration, pct float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	index := int(math.Ceil(pct*float64(len(values)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func startFiberServer(t *testing.T, app *fiber.App) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	done := make(chan struct{})
	go func() {
		if err := app.Listener(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Logf("fiber listener stopped: %v", err)
		}
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	shutdown := func() {
		_ = app.Shutdown()
		_ = listener.Close()
		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
		}
	}

	return "http://" + listener.Addr().String(), shutdown
}

type stubChatService struct{}

func (s *stubChatService) ServeConnection(conn *fiberws.Conn, _ service.ChatConnectionOptions) {
	_ = conn.WriteMessage(fiberws.TextMessage, []byte(`{"type":"welcome"}`))
	_ = conn.Close()
}

func (s *stubChatService) History(context.Context, dto.ChatHistoryQuery) ([]dto.ChatMessageResponse, error) {
	return []dto.ChatMessageResponse{}, nil
}

func (s *stubChatService) Start(context.Context) {}

type stubNotificationService struct{}

func (s *stubNotificationService) Publish(ctx context.Context, payload dto.NotificationCreateRequest) (dto.NotificationResponse, error) {
	return dto.NotificationResponse{ID: 1, UserID: payload.UserID, Type: payload.Type, Message: payload.Message}, nil
}

func (s *stubNotificationService) List(ctx context.Context, userID string, limit, offset int) ([]dto.NotificationResponse, error) {
	return []dto.NotificationResponse{{ID: 1, UserID: userID, Type: "system", Message: "hello", CreatedAt: time.Now(), UpdatedAt: time.Now()}}, nil
}

func (s *stubNotificationService) MarkRead(ctx context.Context, id uint, userID string) (dto.NotificationResponse, error) {
	return dto.NotificationResponse{ID: id, UserID: userID, Type: "system", Message: "hello", Read: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func (s *stubNotificationService) Subscribe(userID string) (<-chan dto.NotificationResponse, func()) {
	ch := make(chan dto.NotificationResponse, 1)
	ch <- dto.NotificationResponse{ID: 99, UserID: userID, Type: "assignment", Message: "graded", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	cleanup := func() { close(ch) }
	return ch, cleanup
}

func (s *stubNotificationService) Start(context.Context) {}
