package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/shinjuwu/TheNuts/internal/auth"
	"github.com/shinjuwu/TheNuts/internal/game/adapter/ws"
	"github.com/shinjuwu/TheNuts/internal/game/domain"
	"github.com/shinjuwu/TheNuts/pkg/di"
	"go.uber.org/zap"
)

func main() {
	// 1. 初始化 App (透過 Wire DI)
	app, err := di.InitApp("config.yaml")
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	// 2. 接線
	app.SessionManager.SetTableNotifier(app.TableManager)
	app.WSHandler.SetAllowedOrigins(app.Config.Server.AllowedOrigins)
	app.TableManager.SetLogger(app.Logger)
	app.TableManager.SetOnSessionChipsUpdate(func(playerID string, chips int64) {
		playerUUID, err := uuid.Parse(playerID)
		if err != nil {
			return
		}
		if session, exists := app.SessionManager.GetSession(playerUUID); exists {
			session.UpdateChips(chips)
		}
	})
	app.TableManager.SetOnTableEvent(func(event domain.TableEvent) {
		resp := ws.Response{
			Type:      string(event.Type),
			Payload:   event.Data,
			Timestamp: time.Now(),
		}
		if event.TargetPlayerID != "" {
			playerUUID, err := uuid.Parse(event.TargetPlayerID)
			if err == nil {
				app.SessionManager.SendToPlayer(playerUUID, resp)
			}
		} else {
			app.SessionManager.BroadcastToTable(event.TableID, resp)
		}
	})

	// 3. 啟動背景服務
	go app.Hub.Run()
	app.Logger.Info("server started",
		zap.String("host", app.Config.Server.Host),
		zap.String("port", app.Config.Server.Port),
	)

	// 3. 設定 HTTP 路由
	mux := http.NewServeMux()

	// 認證路由（公開）
	mux.HandleFunc("/api/auth/register", app.AuthHandler.HandleRegister)
	mux.HandleFunc("/api/auth/login", app.AuthHandler.HandleLogin)

	// 票券路由（需要 JWT 認證）
	jwtMiddleware := auth.JWTMiddleware(app.JWTService)
	mux.Handle("/api/auth/ticket", jwtMiddleware(http.HandlerFunc(app.AuthHandler.HandleGetTicket)))

	// WebSocket 路由（需要票券）
	mux.Handle("/ws", app.WSHandler)

	// 靜態文件服務 (SPA mode)
	// 指向 web/dist
	distDir := "./web/dist"
	fs := http.FileServer(http.Dir(distDir))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API 和 WS 請求不處理
		if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/ws") {
			http.NotFound(w, r)
			return
		}

		// 嘗試查找文件
		path := filepath.Join(distDir, r.URL.Path)
		_, err := os.Stat(path)

		// 如果文件不存在且不是 API/WS，則返回 index.html (SPA Fallback)
		if os.IsNotExist(err) && !strings.Contains(r.URL.Path, ".") {
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}

		fs.ServeHTTP(w, r)
	}))

	srv := &http.Server{
		Addr:    ":" + app.Config.Server.Port,
		Handler: mux,
	}

	// 4. 優雅關閉 (Graceful Shutdown)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Logger.Fatal("listen error", zap.Error(err))
		}
	}()

	// 監聽訊號
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Logger.Info("shutting down server...")

	// 設定關閉超時
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		app.Logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	app.Stop(ctx)
	app.Logger.Info("server exited")
}
