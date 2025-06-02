package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jingdezhen-ceramics-backend/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	userRepo := user.NewRepository(dbPool)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)
	// You'll also need an admin handler if it's separate
	// adminHandler := user.NewAdminHandler(userService, other admin services)

	ceramicStoryRepo := ceramicstory.NewRepository(dbPool)
	ceramicStoryService := ceramicstory.NewService(ceramicStoryRepo)
	ceramicStoryHandler := ceramicstory.NewHandler(ceramicStoryService)

	galleryRepo := gallery.NewRepository(dbPool)
	galleryService := gallery.NewService(galleryRepo) // galleryService might also need e.g. userRepo if favorites involve user data directly in service
	galleryHandler := gallery.NewHandler(galleryService)

	engageRepo := engage.NewRepository(dbPool)
	engageService := engage.NewService(engageRepo)
	engageHandler := engage.NewHandler(engageService)

	courseRepo := course.NewRepository(dbPool)
	courseService := course.NewService(courseRepo)
	courseHandler := course.NewHandler(courseService)

	forumRepo := forum.NewRepository(dbPool)
	forumService := forum.NewService(forumRepo)
	forumHandler := forum.NewHandler(forumService)

	portfolioRepo := portfolio.NewRepository(dbPool)
	portfolioService := portfolio.NewService(portfolioRepo)
	portfolioHandler := portfolio.NewHandler(portfolioService)

	contactService := contact.NewService( /* dependencies like an emailer service */ )
	contactHandlerInstance := contact.NewHandler(contactService)

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
		contactHandler,
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
