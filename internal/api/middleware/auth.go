package middleware

import (
	"jingdezhen-ceramics-backend/internal/models"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// JWTMAuth configures and returns EchoFiber's JWT middleware.
// It uses the jwtSecretKey from the config file (.env).
func JWTMAuth(jwtSecretKey string) fiber.Handler {
	return jwtware.New(jwtware.Config{
		// SigningKey is the secret key used to verify the JWT's signature.
		SigningKey: jwtware.SigningKey{Key: []byte(jwtSecretKey)},
		// ContextKey is the key used to store the token in c.Locals(). Default is "user".
		ContextKey: "user",
		// SuccessHandler is called after a token is successfully validated.
		// We use it here to extract our custom claims and put them into the context.
		SuccessHandler: func(c *fiber.Ctx) error {
			// Get the token from c.Locals("user")
			token := c.Locals("user").(*jwt.Token)
			// Type-assert the claims to our custom claims struct.
			claims := token.Claims.(jwt.MapClaims)

			// Set our specific claims into c.Locals for easy access in subsequent handlers.
			c.Locals("userID", claims["user_id"])
			c.Locals("userEmail", claims["email"])
			c.Locals("userRole", claims["role"])

			// Crucially, we must call c.Next() to pass control to the next handler.
			return c.Next()
		},
		// ErrorHandler is called when there's an error in token validation.
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Return a clear error message to the client.
			if err.Error() == "Missing or malformed JWT" {
				return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: "Missing or malformed JWT"})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{Message: "Invalid or expired JWT"})
		},
	})
}

// AdminRequired is a middleware that checks if the authenticated user has an 'admin' role.
// It must be used AFTER the JWTMAuth middleware in the chain.
func AdminRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the userRole from c.Locals, which was set by JWTMAuth's SuccessHandler.
		userRole, ok := c.Locals("userRole").(string)
		if !ok {
			// This case might happen if AdminRequired is mistakenly used without JWTMAuth.
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{Message: "Permission denied: Role not determined"})
		}

		if userRole != models.RoleAdmin {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{Message: "Admin access required"})
		}

		// If the role is correct, proceed to the next handler.
		return c.Next()
	}
}

// NormalUserRequired is a middleware that checks if the authenticated user has a 'normal_user' role.
// Admins are also considered to satisfy this requirement.
// It must be used AFTER the JWTMAuth middleware.
func NormalUserRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("userRole").(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{Message: "Permission denied: Role not determined"})
		}

		// Admins are often considered to also satisfy "normal user" requirements.
		if userRole != models.RoleNormalUser && userRole != models.RoleAdmin {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{Message: "Access restricted to normal users"})
		}

		return c.Next()
	}
}
