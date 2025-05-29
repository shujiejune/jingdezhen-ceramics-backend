package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	//"github.com/labstack/echo/v4/middleware" // if using echo's built-in JWT
)

// Claims struct for JWT
type JwtCustomClaims struct {
	UserID string `json:"user_id"` // Or int, depending on your user ID type
	Email  string `json:"email"`
	Role   string `json:"role"` // "admin", "normal_user"
	jwt.RegisteredClaims
}

// JWTMAuth middleware validates Supabase JWT
// You'll need your Supabase JWT Secret
func JWTMAuth(jwtSecretKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Missing Authorization Header"})
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid Authorization Header format"})
			}
			tokenString := parts[1]

			claims := &JwtCustomClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				// Validate the alg is what you expect:
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Unexpected signing method: %v", token.Header["alg"]))
				}
				return []byte(jwtSecretKey), nil
			})

			if err != nil {
				if err == jwt.ErrSignatureInvalid {
					return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid token signature"})
				}
				// Check for expired token specifically if using RegisteredClaims
				if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
					return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Token expired"})
				}
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid token", "error": err.Error()})
			}

			if !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid token"})
			}

			// Store user info in context for handlers to access
			c.Set("userID", claims.UserID)
			c.Set("userEmail", claims.Email)
			c.Set("userRole", claims.Role) // Make sure Supabase JWT includes role or derive it

			return next(c)
		}
	}
}

// AdminRequired middleware checks if the user has an 'admin' role.
// Should be used AFTER JWTMAuth.
func AdminRequired() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole, ok := c.Get("userRole").(string)
			if !ok || userRole != "admin" { // Assuming "admin" is the role string
				return c.JSON(http.StatusForbidden, map[string]string{"message": "Admin access required"})
			}
			return next(c)
		}
	}
}
