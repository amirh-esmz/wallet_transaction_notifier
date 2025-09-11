package httpserver

import (
    "net/http"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/labstack/echo/v4"

    "github.com/you/wallet_transaction_notifier/internal/config"
    "github.com/you/wallet_transaction_notifier/internal/services"
)

// HealthHandler returns service health.
func HealthHandler(c echo.Context) error {
    return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// LoginHandler issues JWT tokens for demo purposes only.
func LoginHandler(cfg config.Config) echo.HandlerFunc {
    return func(c echo.Context) error {
        // In real-world, validate credentials. Here we accept any.
        userID := c.QueryParam("userId")
        if userID == "" {
            userID = "u-demo"
        }
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
            "sub": userID,
            "exp": time.Now().Add(24 * time.Hour).Unix(),
        })
        s, err := token.SignedString([]byte(cfg.JWTSecret))
        if err != nil {
            return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
        }
        return c.JSON(http.StatusOK, map[string]string{"token": s})
    }
}

// ListWalletsHandler calls API service to list wallets.
func ListWalletsHandler(api *services.APIService) echo.HandlerFunc {
    return func(c echo.Context) error {
        userID := c.Param("userId")
        items, err := api.ListUserWallets(c.Request().Context(), userID)
        if err != nil {
            return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
        }
        return c.JSON(http.StatusOK, items)
    }
}


