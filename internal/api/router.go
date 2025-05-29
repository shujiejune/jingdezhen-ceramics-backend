package api

import (
	"jingdezhen-ceramics-backend/internal/api/middleware"
	"jingdezhen-ceramics-backend/internal/ceramicstory"
	"jingdezhen-ceramics-backend/internal/course"
	"jingdezhen-ceramics-backend/internal/engage"
	"jingdezhen-ceramics-backend/internal/forum"
	"jingdezhen-ceramics-backend/internal/gallery"
	"jingdezhen-ceramics-backend/internal/portfolio"
	"jingdezhen-ceramics-backend/internal/user"
	"net/http"

	//"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// SetupRoutes configures the API routes.
// You'll pass instantiated handlers or dependencies like dbPool here.
func SetupRoutes(e *echo.Echo, jwtSecretKey string /*, db *pgxpool.Pool - example dep */) {

	// Placeholder: Initialize dependencies (in a real app, these come from main.go or a dep injection container)
	// This is a simplified example; proper DI is recommended.
	// For actual implementation, repositories and services should be initialized in main and passed down.
	// Here, handlers might instantiate their own services/repos or receive them.

	// --- Public Routes ---
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome to Jingdezhen Ceramic Culture API!")
	})
	e.POST("/contact", func(c echo.Context) error { // Example contact route
		// Handle contact form submission
		return c.JSON(http.StatusOK, map[string]string{"message": "Message received!"})
	})

	// Auth routes (if not fully handled by Supabase client-side, e.g., a route to trigger something after Supabase auth)
	// Or if you implement parts of user management for admin purposes not directly covered by Supabase client.
	// user.RegisterAuthRoutes(e.Group("/auth"), userService) // Assuming userService is available

	// --- User Profile (Protected) ---
	// User handler initialization (example, assuming it takes a JWT secret for now)
	userHandler := user.NewHandler( /* userService */ ) // Pass actual service
	profileGroup := e.Group("/profile")
	profileGroup.Use(middleware.JWTMAuth(jwtSecretKey)) // Your JWT Auth Middleware
	{
		profileGroup.GET("", userHandler.GetProfile)
		profileGroup.PUT("", userHandler.UpdateProfile)
		profileGroup.GET("/notes", userHandler.GetUserNotes)
		profileGroup.POST("/notes", userHandler.CreateUserNote)
		// ... other profile routes (favorites, subscriptions, inbox)
	}

	// --- Ceramic Story (Public) ---
	ceramicStoryHandler := ceramicstory.NewHandler( /* ceramicStoryService */ )
	csGroup := e.Group("/ceramicstory")
	{
		csGroup.GET("", ceramicStoryHandler.GetAllDynasties) // Get all dynasty stories
		// csGroup.GET("/:dynasty_id", ceramicStoryHandler.GetDynasty) // If you need individual
	}

	// --- Gallery (Public for viewing, Protected for actions) ---
	galleryHandler := gallery.NewHandler( /* galleryService */ )
	gGroup := e.Group("/gallery")
	{
		gGroup.GET("", galleryHandler.GetArtworks) // Params: ?category=...&artist=...
		gGroup.GET("/:artwork_id", galleryHandler.GetArtworkByID)
		gGroup.POST("/:artwork_id/favorite", galleryHandler.MarkAsFavorite, middleware.JWTMAuth(jwtSecretKey))
		gGroup.DELETE("/:artwork_id/favorite", galleryHandler.UnmarkAsFavorite, middleware.JWTMAuth(jwtSecretKey))
		gGroup.POST("/:artwork_id/notes", galleryHandler.AddNoteToArtwork, middleware.JWTMAuth(jwtSecretKey))
	}

	// --- Engage (Public) ---
	engageHandler := engage.NewHandler( /* engageService */ )
	engageGroup := e.Group("/engage")
	{
		engageGroup.GET("", engageHandler.GetActivities)
		engageGroup.GET("/:activity_id_or_slug", engageHandler.GetActivityArticle) // For detailed article
	}

	// --- Course (Mixed Public/Protected) ---
	courseHandler := course.NewHandler( /* courseService */ )
	cGroup := e.Group("/courses")
	{
		cGroup.GET("", courseHandler.GetAllCourses)
		cGroup.GET("/:course_id", courseHandler.GetCourseDetails)                       // Chapters list
		cGroup.GET("/:course_id/chapters/:chapter_id", courseHandler.GetChapterContent) // Public up to chapter 2
		// Protected access for full course and progress:
		cGroup.POST("/:course_id/chapters/:chapter_id/progress", courseHandler.UpdateProgress, middleware.JWTMAuth(jwtSecretKey))
		cGroup.POST("/:course_id/chapters/:chapter_id/notes", courseHandler.AddNoteToChapter, middleware.JWTMAuth(jwtSecretKey))
		// Video related endpoints if needed (e.g., video quiz submissions)
	}

	// --- Forum (Public read, Protected write/interact) ---
	forumHandler := forum.NewHandler( /* forumService */ )
	fGroup := e.Group("/forum")
	{
		fGroup.GET("/posts", forumHandler.GetPosts)           // Params: ?page=1&limit=10&sort=latest|hottest&tag=...&category=...
		fGroup.GET("/posts/search", forumHandler.SearchPosts) // Param: ?q=keyword
		fGroup.GET("/posts/:post_id", forumHandler.GetPostByID)
		fGroup.GET("/topics", forumHandler.GetTopicsTagCloud) // Tag cloud
		fGroup.GET("/categories", forumHandler.GetCategories)

		// Protected actions
		authForumGroup := fGroup.Group("") // Create a sub-group for auth middleware on specific routes
		authForumGroup.Use(middleware.JWTMAuth(jwtSecretKey))
		{
			authForumGroup.POST("/posts", forumHandler.CreatePost)
			authForumGroup.PUT("/posts/:post_id", forumHandler.UpdatePost)    // Check ownership
			authForumGroup.DELETE("/posts/:post_id", forumHandler.DeletePost) // Check ownership or admin
			authForumGroup.POST("/posts/:post_id/comments", forumHandler.CreateComment)
			authForumGroup.POST("/posts/:post_id/like", forumHandler.LikePost)
			authForumGroup.POST("/posts/:post_id/save", forumHandler.SavePost)
			// ... comment likes, saves, updates, deletes
		}
	}

	// --- Portfolio (Public read, Protected kudos) ---
	portfolioHandler := portfolio.NewHandler( /* portfolioService */ )
	pGroup := e.Group("/portfolio")
	{
		pGroup.GET("", portfolioHandler.GetWorks) // Params: ?page=1&category=...&sort=kudos
		pGroup.GET("/:work_id", portfolioHandler.GetWorkByID)
		pGroup.POST("/:work_id/kudos", portfolioHandler.LeaveKudo, middleware.JWTMAuth(jwtSecretKey))
	}

	// --- Admin Routes (Protected by Admin Role) ---
	adminHandler := user.NewAdminHandler( /* userService, other admin services */ ) // A specific admin handler
	adminGroup := e.Group("/admin")
	adminGroup.Use(middleware.JWTMAuth(jwtSecretKey), middleware.AdminRequired()) // JWT + Admin Role Check
	{
		adminGroup.GET("/dashboard/student-progress", adminHandler.GetStudentProgressDashboard)
		adminGroup.POST("/forum/posts/:post_id/archive", adminHandler.ArchiveForumPost)
		adminGroup.DELETE("/forum/posts/:post_id", adminHandler.DeleteForumPostAsAdmin)
		adminGroup.POST("/portfolio/works/:work_id/highlight", adminHandler.HighlightPortfolioWork)
		// ... other admin functionalities
	}
}

// Define your handlers (e.g., ceramicstory.Handler, user.Handler) in their respective packages.
// Each handler would have methods like:
// func (h *MyHandler) GetData(c echo.Context) error { ... }
