package api

import (
	"jingdezhen-ceramics-backend/internal/api/middleware"
	"jingdezhen-ceramics-backend/internal/modules/ceramicstory"
	"jingdezhen-ceramics-backend/internal/modules/course"
	"jingdezhen-ceramics-backend/internal/modules/engage"
	"jingdezhen-ceramics-backend/internal/modules/forum"
	"jingdezhen-ceramics-backend/internal/modules/gallery"
	"jingdezhen-ceramics-backend/internal/modules/note"
	"jingdezhen-ceramics-backend/internal/modules/notification"
	"jingdezhen-ceramics-backend/internal/modules/portfolio"
	"jingdezhen-ceramics-backend/internal/modules/user"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures the API routes.
func SetupRoutes(
	app *fiber.App, jwtSecretKey string,
	userHandler *user.Handler,
	notifHandler *notification.Handler,
	forumHandler *forum.Handler,
	noteHandler *note.Handler,
	csHandler *ceramicstory.Handler,
	galleryHandler *gallery.Handler,
	engageHandler *engage.Handler,
	courseHandler *course.Handler,
	portfolioHandler *portfolio.Handler,
) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Welcome to Jingdezhen Ceramics Learning and Communication Platform!"})
	})

	app.Get("/ws", websocket.New(wsHandler))

	/* --- Contact (send feedback) --- */
	app.Post("/contact", userHandler.SubmitContactForm)

	/* --- Auth (Public) --- */
	authGroup := app.Group("/auth")
	{
		authGroup.Post("/signup", userHandler.Signup)
		authGroup.Post("/login", userHandler.Login)
		authGroup.Post("/activate", userHandler.ActivateAccount)
		authGroup.Post("resend-activation", userHandler.ResendActivation)
		authGroup.Post("request-password-reset", userHandler.RequestPasswordReset)
		authGroup.Post("reset-password", userHandler.ResetPassword)
		authGroup.Get("/google/login", userHandler.GoogleLogin)
		authGroup.Get("/google/callback", userHandler.GoogleCallback)
	}

	/* --- User Profile (Protected) --- */
	// If need backend routes for auth (e.g., refresh token, logout initiated by backend), define here.
	profileGroup := app.Group("/profile")
	profileGroup.Use(middleware.JWTMAuth(jwtSecretKey))
	{
		profileGroup.Get("", userHandler.GetProfile)
		profileGroup.Put("", userHandler.UpdateProfile)
		// ... other user-specific routes like badges, subscriptions
	}

	/* --- Notification Module (Protected) --- */
	notifGroup := app.Group("/notifications")
	notifGroup.Use(middleware.JWTMAuth(jwtSecretKey))
	{
		notifGroup.Get("", notifHandler.GetNotifications)
		notifGroup.Get("/unread-count", notifHandler.GetUnreadNotificationCount)
		notifGroup.Post("/mark-all-read", notifHandler.MarkAllAsRead)
		notifGroup.Post("/:notification_id/mark-read", notifHandler.MarkAsRead)
	}

	/* --- Forum (Public read, Protected write/interact) --- */
	fGroup := app.Group("/forum")
	{
		fGroup.Get("/posts", forumHandler.GetPosts)           // Params: ?page=1&limit=10&sort=latest|hottest&tag=...&category=...
		fGroup.Get("/posts/search", forumHandler.SearchPosts) // Param: ?q=keyword
		fGroup.Get("/posts/:post_id", forumHandler.GetPostByID)
		fGroup.Get("/topics", forumHandler.GetTopicsTagCloud) // Tag cloud
		fGroup.Get("/categories", forumHandler.GetCategories)

		// Protected actions
		authForumGroup := fGroup.Group("")
		authForumGroup.Use(middleware.JWTMAuth(jwtSecretKey))
		{
			authForumGroup.Get("/saved-posts", forumHandler.GetSavedForumPosts)
			authForumGroup.Post("/posts", forumHandler.CreatePost)
			authForumGroup.Put("/posts/:post_id", forumHandler.UpdatePost)    // Check ownership
			authForumGroup.Delete("/posts/:post_id", forumHandler.DeletePost) // Check ownership or admin
			authForumGroup.Post("/posts/:post_id/comments", forumHandler.CreateComment)
			authForumGroup.Post("/posts/:post_id/comments/:comment_id/replies", forumHandler.CreateReply)
			authForumGroup.Put("/comments/:comment_id", forumHandler.UpdateComment)
			authForumGroup.Delete("/comments/:comment_id", forumHandler.DeleteComment)
			authForumGroup.Post("/posts/:post_id/like", forumHandler.TogglePostLike)
			authForumGroup.Post("/posts/:post_id/save", forumHandler.TogglePostSave)
			authForumGroup.Post("/comments/:comment_id/like", forumHandler.ToggleCommentLike)
		}
	}

	/* --- Note Module (Protected) --- */
	noteGroup := app.Group("/notes")
	noteGroup.Use(middleware.JWTMAuth(jwtSecretKey))
	{
		noteGroup.Get("", noteHandler.GetUserNotes)
		noteGroup.Get("/:note_id", noteHandler.GetUserNoteByID)
		noteGroup.Post("", noteHandler.CreateUserNote)
		noteGroup.Put("/:note_id", noteHandler.UpdateUserNote)
		noteGroup.Delete("/:note_id", noteHandler.DeleteUserNote)
		noteGroup.Post("/:note_id/links", noteHandler.AddLinkToNote)
		noteGroup.Delete("/:note_id/links/:link_id", noteHandler.RemoveLinkFromNote)
		noteGroup.Post("/:note_id/publish-to-forum", noteHandler.PublishNoteToForum)
	}

	/* --- Ceramic Story (Public) --- */
	csGroup := app.Group("/ceramicstory")
	{
		csGroup.Get("", csHandler.GetAllDynasties)
		csGroup.Get("/:dynasty_id_or_slug", csHandler.GetDynastyDetail)
	}

	/* --- Gallery (Public for viewing, Protected for actions) --- */
	gGroup := app.Group("/gallery")
	{
		gGroup.Get("/artworks", galleryHandler.GetArtworks) // Params: ?category=...&artist=...
		gGroup.Get("/artworks/:artwork_id", galleryHandler.GetArtworkByID)
		gGroup.Get("/artists", galleryHandler.GetArtists)
		gGroup.Get("/artists/:artist_id", galleryHandler.GetArtistByID)
		gGroup.Get("/categories", galleryHandler.GetGalleryCategories)

		// Protected actions for gallery
		authGalleryGroup := gGroup.Group("")
		authGalleryGroup.Use(middleware.JWTMAuth(jwtSecretKey))
		{
			authGalleryGroup.Get("/favorites", galleryHandler.GetFavoriteArtworks)
			authGalleryGroup.Post("/artworks/:artwork_id/favorite", galleryHandler.MarkAsFavorite)
			authGalleryGroup.Delete("/artworks/:artwork_id/favorite", galleryHandler.UnmarkAsFavorite)
			authGalleryGroup.Post("/artworks/:artwork_id/notes", galleryHandler.AddNoteToArtwork)
		}
	}

	/* --- Engage (Public) --- */
	engageGroup := app.Group("/engage")
	{
		engageGroup.Get("", engageHandler.GetActivities)
		engageGroup.Get("/:activity_id_or_slug", engageHandler.GetActivityArticle) // For detailed article
	}

	/* --- Course (Mixed Public/Protected) --- */
	cGroup := app.Group("/courses")
	{
		cGroup.Get("", courseHandler.GetAllCourses)
		cGroup.Get("/quizzes/:quiz_id", courseHandler.GetQuiz)
		cGroup.Get("/:course_id", courseHandler.GetCourseDetails)                       // Chapters list
		cGroup.Get("/:course_id/chapters/:chapter_id", courseHandler.GetChapterContent) // Public up to chapter 2

		// Protected access for full course and progress:
		authCourseGroup := cGroup.Group("")
		authCourseGroup.Use(middleware.JWTMAuth(jwtSecretKey))
		authCourseGroup.Use(middleware.NormalUserRequired())
		{
			authCourseGroup.Post("/:course_id/enroll", courseHandler.EnrollCourse)
			authCourseGroup.Get("/enrolled", courseHandler.GetEnrolledCourses)
			authCourseGroup.Get("/:course_id/chapters/:chapter_id/full", courseHandler.GetFullChapterContentForEnrolled)
			authCourseGroup.Post("/:course_id/chapters/:chapter_id/blocks/:block_id/complete", courseHandler.MarkContentBlockComplete)
			authCourseGroup.Post("/:course_id/chapters/:chapter_id/blocks/:block_id/video-progress", courseHandler.UpdateVideoProgress)
			authCourseGroup.Post("/:course_id/chapters/:chapter_id/notes", courseHandler.AddNoteToChapter)
			authCourseGroup.Post("/:course_id/chapters/:chapter_id/quizzes/:assignment_id/submit", courseHandler.SubmitAssignment)
			authCourseGroup.Post("/:course_id/chapters/:chapter_id/quizzes/:quiz_id/submit", courseHandler.SubmitQuiz)
			//authCourseGroup.POST("/:course_id/chapters/:chapter_id/video-quizzes/:quiz_id/submit", courseHandler.SubmitVideoQuiz)
		}
	}

	/* --- Portfolio (Public read, Protected kudos) --- */
	pGroup := app.Group("/portfolio")
	{
		pGroup.Get("", portfolioHandler.GetWorks) // Params: ?page=1&category=...&sort=upvotes
		pGroup.Get("/:work_id", portfolioHandler.GetWorkByID)

		// Protected actions
		authPortfolioGroup := pGroup.Group("")
		authPortfolioGroup.Use(middleware.JWTMAuth(jwtSecretKey))
		{
			authPortfolioGroup.Post("/works", portfolioHandler.CreateWork)
			authPortfolioGroup.Put("/works/:work_id", portfolioHandler.UpdateWork)
			authPortfolioGroup.Delete("/works/:work_id", portfolioHandler.DeleteWork)
			authPortfolioGroup.Post("/works/:work_id/upvotes", portfolioHandler.ToggleWorkUpvote)
		}
	}

	/* --- Admin Routes (Protected by Admin Role) --- */
	adminGroup := app.Group("/admin")
	adminGroup.Use(middleware.JWTMAuth(jwtSecretKey))
	adminGroup.Use(middleware.AdminRequired())
	{
		adminGroup.Get("/dashboard/student-progress", adminHandler.GetStudentProgressDashboard)
		adminGroup.Post("/forum/posts/:post_id/pin", adminHandler.PinForumPost)
		adminGroup.Post("/forum/posts/:post_id/archive", adminHandler.ArchiveForumPost)
		adminGroup.Delete("/forum/posts/:post_id", adminHandler.DeleteForumPostAsAdmin)
		adminGroup.Post("/portfolio/works/:work_id/highlight", adminHandler.HighlightPortfolioWork)
		// ... other admin functionalities
	}
}
