package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jingdezhen-ceramics-backend/internal/api"
	"jingdezhen-ceramics-backend/internal/config"
	"jingdezhen-ceramics-backend/internal/modules/ceramicstory"
	"jingdezhen-ceramics-backend/internal/modules/course"
	"jingdezhen-ceramics-backend/internal/modules/engage"
	"jingdezhen-ceramics-backend/internal/modules/forum"
	"jingdezhen-ceramics-backend/internal/modules/gallery"
	"jingdezhen-ceramics-backend/internal/modules/note"
	"jingdezhen-ceramics-backend/internal/modules/notification"
	"jingdezhen-ceramics-backend/internal/modules/portfolio"
	"jingdezhen-ceramics-backend/internal/modules/user"
	"jingdezhen-ceramics-backend/pkg/email"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Could not load config: %v", err)
	}

	e := echo.New()
	e.Logger.Fatal(e.Start(":1323"))

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{ // Configure CORS appropriately
		AllowOrigins: []string{"http://localhost:5173", cfg.ClientOrigin}, // Your SvelteKit dev and prod origins
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Database connection
	dbConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to parse database configuration: %v\n", err)
	}

	dbPool, err := pgxpool.NewWithConfig(context.Background(), dbConfig)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}
	defer dbPool.Close() // Ensure pool is closed when main exits

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}
	e.Logger.Info("Successfully connected to the database!")

	// Dependency injection
	// 1. Initialize Google OAuth Config
	googleOAuthConfig := &oauth2.Config{
		RedirectURL:  cfg.GoogleOAuthRedirectURL,
		ClientID:     cfg.GoogleOAuthClientID,
		ClientSecret: cfg.GoogleOAuthClientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	// 2. Initialize other services
	sesSender, err := email.NewSESV2Sender(context.Background(), cfg.AWSRegion, cfg.AdminEmail)
	if err != nil {
		log.Fatalf("Failed to create SES sender: %v", err)
	}
	templateManager, err := email.NewTemplateManager()
	if err != nil {
		log.Fatalf("Failed to parse email templates: %v", err)
	}

	forumRepo := forum.NewRepository(dbPool)
	forumService := forum.NewService(forumRepo)
	forumHandler := forum.NewHandler(forumService)

	userRepo := user.NewRepository(dbPool)
	userService := user.NewService(
		userRepo,
		forumService,
		sesSender,
		templateManager,
		cfg.JWTSecret,
		cfg.ClientOrigin,
		cfg.AdminEmail,
		googleOAuthConfig,
	)
	userHandler := user.NewHandler(userService)
	// You'll also need an admin handler if it's separate
	// adminHandler := user.NewAdminHandler(userService, other admin services)

	ceramicStoryRepo := ceramicstory.NewRepository(dbPool)
	ceramicStoryService := ceramicstory.NewService(ceramicStoryRepo)
	ceramicStoryHandler := ceramicstory.NewHandler(ceramicStoryService)

	galleryRepo := gallery.NewRepository(dbPool)
	galleryService := gallery.NewService(galleryRepo, userService)
	galleryHandler := gallery.NewHandler(galleryService)

	engageRepo := engage.NewRepository(dbPool)
	engageService := engage.NewService(engageRepo)
	engageHandler := engage.NewHandler(engageService)

	courseRepo := course.NewRepository(dbPool)
	courseService := course.NewService(courseRepo, userService)
	courseHandler := course.NewHandler(courseService)

	portfolioRepo := portfolio.NewRepository(dbPool)
	portfolioService := portfolio.NewService(portfolioRepo)
	portfolioHandler := portfolio.NewHandler(portfolioService)

	// Initialize router, passing all handlers and other necessary dependencies
	api.SetupRoutes(e, cfg.JWTSecret,
		userHandler,
		// adminHandler, // Pass if you have a separate admin handler instance
		ceramicStoryHandler,
		galleryHandler,
		engageHandler,
		courseHandler,
		forumHandler,
		portfolioHandler,
	)

	// Start server (graceful shutdown logic)
	go func() {
		if err := e.Start(":" + cfg.ServerPort); err != nil && err != http.ErrServerClosed {
			// Using e.Logger here because 'e' is initialized.
			e.Logger.Fatal("shutting down the server an error occurred:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exiting")
}
