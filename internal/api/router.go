package api

import (
	"jingdezhen-ceramics-backend/internal/api/middleware"
	"jingdezhen-ceramics-backend/internal/ceramicstory"
	"jingdezhen-ceramics-backend/internal/contact"
	"jingdezhen-ceramics-backend/internal/course"
	"jingdezhen-ceramics-backend/internal/engage"
	"jingdezhen-ceramics-backend/internal/forum"
	"jingdezhen-ceramics-backend/internal/gallery"
	"jingdezhen-ceramics-backend/internal/portfolio"
	"jingdezhen-ceramics-backend/internal/user"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SetupRoutes configures the API routes.
func SetupRoutes(
	e *echo.Echo, jwtSecretKey string,
	userHandler *user.Handler,
	adminHandler *user.AdminHandler,
	csHandler *ceramicstory.Handler,
	galleryHandler *gallery.Handler,
	engageHandler *engage.Handler,
	courseHandler *course.Handler,
	forumHandler *forum.Handler,
	portfolioHandler *portfolio.Handler,
	contactHandler *contact.Handler,
) {
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Welcome to Jingdezhen Ceramics Platform!"})
	})

	/* --- User Profile (Protected) --- */
	// If need backend routes for auth (e.g., refresh token, logout initiated by backend), define here.
	profileGroup := e.Group("/profile")
	profileGroup.Use(middleware.JWTMAuth(jwtSecretKey))
	{
		profileGroup.GET("", userHandler.GetProfile)
		profileGroup.PUT("", userHandler.UpdateProfile)
		profileGroup.GET("/notifications", userHandler.GetNotifications)
		profileGroup.GET("/notes", userHandler.GetUserNotes)
		profileGroup.POST("/notes", userHandler.CreateUserNote)
		profileGroup.PUT("/notes/:note_id", userHandler.UpdateUserNote)
		profileGroup.DELETE("/notes/:note_id", userHandler.DeleteUserNote)
		profileGroup.GET("/favorite-artworks", userHandler.GetFavoriteArtworks)
		profileGroup.GET("/saved-posts", userHandler.GetSavedForumPosts)
		// ... other user-specific routes like badges, subscriptions
	}

	/* --- Ceramic Story (Public) --- */
	csGroup := e.Group("/ceramicstory")
	{
		csGroup.GET("", csHandler.GetAllDynasties)
		csGroup.GET("/:dynasty_id_or_slug", csHandler.GetDynastyDetail)
	}

	/* --- Gallery (Public for viewing, Protected for actions) --- */
	gGroup := e.Group("/gallery")
	{
		gGroup.GET("/artworks", galleryHandler.GetArtworks) // Params: ?category=...&artist=...
		gGroup.GET("/artworks/:artwork_id", galleryHandler.GetArtworkByID)
		gGroup.GET("/artists", galleryHandler.GetArtists)
		gGroup.GET("/artists/:artist_id", galleryHandler.GetArtistByID)
		gGroup.GET("/categories", galleryHandler.GetGalleryCategories)

		// Protected actions for gallery
		authGalleryGroup := gGroup.Group("")
		authGalleryGroup.Use(middleware.JWTMAuth(jwtSecretKey))
		{
			authGalleryGroup.POST("/artworks/:artwork_id/favorite", galleryHandler.MarkAsFavorite)
			authGalleryGroup.DELETE("/artworks/:artwork_id/favorite", galleryHandler.UnmarkAsFavorite)
			authGalleryGroup.POST("/artworks/:artwork_id/notes", galleryHandler.AddNoteToArtwork)
		}
	}

	/* --- Engage (Public) --- */
	engageGroup := e.Group("/engage")
	{
		engageGroup.GET("", engageHandler.GetActivities)
		engageGroup.GET("/:activity_id_or_slug", engageHandler.GetActivityArticle) // For detailed article
	}

	/* --- Course (Mixed Public/Protected) --- */
	cGroup := e.Group("/courses")
	{
		cGroup.GET("", courseHandler.GetAllCourses)
		cGroup.GET("/:course_id", courseHandler.GetCourseDetails)                       // Chapters list
		cGroup.GET("/:course_id/chapters/:chapter_id", courseHandler.GetChapterContent) // Public up to chapter 2

		// Protected access for full course and progress:
		authCourseGroup := cGroup.Group("")
		authCourseGroup.Use(middleware.JWTMAuth(jwtSecretKey))
		authCourseGroup.Use(middleware.NormalUserRequired())
		{
			authCourseGroup.POST("/:course_id/enroll", courseHandler.EnrollCourse)
			authCourseGroup.GET("/:course_id/chapters/:chapter_id/full", courseHandler.GetFullChapterContentForEnrolled)
			authCourseGroup.POST("/:course_id/chapters/:chapter_id/progress", courseHandler.UpdateProgress)
			authCourseGroup.POST("/:course_id/chapters/:chapter_id/notes", courseHandler.AddNoteToChapter)
			authCourseGroup.POST("/:course_id/chapters/:chapter_id/quizzes/:quiz_id/submit", courseHandler.SubmitQuiz)
		}
		// Video related endpoints if needed (e.g., video quiz submissions)
	}

	/* --- Forum (Public read, Protected write/interact) --- */
	fGroup := e.Group("/forum")
	{
		fGroup.GET("/posts", forumHandler.GetPosts)           // Params: ?page=1&limit=10&sort=latest|hottest&tag=...&category=...
		fGroup.GET("/posts/search", forumHandler.SearchPosts) // Param: ?q=keyword
		fGroup.GET("/posts/:post_id", forumHandler.GetPostByID)
		fGroup.GET("/topics", forumHandler.GetTopicsTagCloud) // Tag cloud
		fGroup.GET("/categories", forumHandler.GetCategories)

		// Protected actions
		authForumGroup := fGroup.Group("")
		authForumGroup.Use(middleware.JWTMAuth(jwtSecretKey))
		{
			authForumGroup.POST("/posts", forumHandler.CreatePost)
			authForumGroup.PUT("/posts/:post_id", forumHandler.UpdatePost)    // Check ownership
			authForumGroup.DELETE("/posts/:post_id", forumHandler.DeletePost) // Check ownership or admin
			authForumGroup.POST("/posts/:post_id/comments", forumHandler.CreateComment)
			authForumGroup.PUT("/comments/:comment_id", forumHandler.UpdateComment)
			authForumGroup.DELETE("/comments/:comment_id", forumHandler.DeleteComment)
			authForumGroup.POST("/posts/:post_id/like", forumHandler.LikePost)
			authForumGroup.POST("/posts/:post_id/save", forumHandler.SavePost)
			authForumGroup.POST("/comments/:comment_id/like", forumHandler.LikeComment)
			// ... comment likes, saves, updates, deletes
		}
	}

	/* --- Portfolio (Public read, Protected kudos) --- */
	pGroup := e.Group("/portfolio")
	{
		pGroup.GET("", portfolioHandler.GetWorks) // Params: ?page=1&category=...&sort=kudos
		pGroup.GET("/:work_id", portfolioHandler.GetWorkByID)

		// Protected actions
		authPortfolioGroup := pGroup.Group("")
		authPortfolioGroup.Use(middleware.JWTMAuth(jwtSecretKey))
		{
			authPortfolioGroup.POST("/works", portfolioHandler.CreateWork)
			authPortfolioGroup.PUT("/works/:work_id", portfolioHandler.UpdateWork)
			authPortfolioGroup.DELETE("/works/:work_id", portfolioHandler.DeleteWork)
			authPortfolioGroup.POST("/works/:work_id/kudos", portfolioHandler.LeaveKudo)
		}
	}

	/* --- Admin Routes (Protected by Admin Role) --- */
	adminGroup := e.Group("/admin")
	adminGroup.Use(middleware.JWTMAuth(jwtSecretKey))
	adminGroup.Use(middleware.AdminRequired())
	{
		adminGroup.GET("/dashboard/student-progress", adminHandler.GetStudentProgressDashboard)
		adminGroup.POST("/forum/posts/:post_id/pin", adminHandler.PinForumPost)
		adminGroup.POST("/forum/posts/:post_id/archive", adminHandler.ArchiveForumPost)
		adminGroup.DELETE("/forum/posts/:post_id", adminHandler.DeleteForumPostAsAdmin)
		adminGroup.POST("/portfolio/works/:work_id/highlight", adminHandler.HighlightPortfolioWork)
		// ... other admin functionalities
	}

	/* --- Contact (send feedback) --- */
	e.POST("/contact", contactHandler.SubmitContactForm)
}
