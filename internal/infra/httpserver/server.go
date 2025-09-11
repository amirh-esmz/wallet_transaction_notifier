package httpserver

import (
    "context"
    "fmt"
    "log"

    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"

    "github.com/you/wallet_transaction_notifier/internal/config"
    "github.com/you/wallet_transaction_notifier/internal/ports"
    "github.com/you/wallet_transaction_notifier/internal/services"
)

type Server struct {
    cfg    config.Config
    eb     ports.EventBus
    echo   *echo.Echo
    addr   string
    closed bool
    wallets ports.WalletRepository
    api    *services.APIService
}

func NewServer(cfg config.Config, eb ports.EventBus, wallets ports.WalletRepository) *Server {
    e := echo.New()
    e.HideBanner = true
    e.Use(middleware.Recover())
    e.Use(middleware.Logger())
    e.Use(middleware.JWTWithConfig(middleware.JWTConfig{
        SigningKey: []byte(cfg.JWTSecret),
        Skipper: func(c echo.Context) bool {
            // Allow health and auth endpoints without JWT
            path := c.Path()
            if path == "/health" || path == "/auth/login" {
                return true
            }
            return false
        },
        ContextKey: "user",
        TokenLookup: "header:Authorization:Bearer ",
    }))

    s := &Server{
        cfg:  cfg,
        eb:   eb,
        echo: e,
        addr: fmt.Sprintf(":%s", cfg.AppPort),
        wallets: wallets,
    }
    s.api = services.NewAPIService(wallets)
    s.registerRoutes()
    return s
}

func (s *Server) registerRoutes() {
    s.echo.GET("/health", HealthHandler)
    s.echo.POST("/auth/login", LoginHandler(s.cfg))
    s.echo.GET("/users/:userId/wallets", ListWalletsHandler(s.api))
}

func (s *Server) Start() error {
    log.Printf("API listening on %s", s.addr)
    return s.echo.Start(s.addr)
}

func (s *Server) Stop(ctx context.Context) error {
    if s.closed {
        return nil
    }
    s.closed = true
    return s.echo.Shutdown(ctx)
}


