package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"jingdezhen-ceramics-backend/internal/models"
	"jingdezhen-ceramics-backend/internal/modules/forum"
	emailSvc "jingdezhen-ceramics-backend/pkg/email"
	"jingdezhen-ceramics-backend/pkg/utils"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

// ServiceInterface defines methods for user business logic.
type ServiceInterface interface {
	GetClientOrigin() string

	Signup(ctx context.Context, req models.SignupRequest) (*models.User, error)
	ActivateUserAndLogin(ctx context.Context, token string) (*models.AuthResponse, error)
	ResendActivationEmail(ctx context.Context, email string) error
	Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error)
	HandleGoogleLogin() (string, string, error)
	HandleGoogleCallback(ctx context.Context, code string) (*models.AuthResponse, error)
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token string, newPassword string) (*models.AuthResponse, error)

	GetUserProfile(ctx context.Context, userID string) (*models.User, error)
	UpdateUserProfile(ctx context.Context, userID string, data models.UserUpdateData) (*models.User, error)
	HandleContactSubmission(ctx context.Context, data models.ContactFormData) error

	// User Notes
	ListUserNotes(ctx context.Context, userID string, page, limit int) ([]models.UserNote, int, error)
	GetUserNoteDetails(ctx context.Context, userID string, noteID int) (*models.UserNote, error)
	CreateUserNote(ctx context.Context, userID string, data models.CreateUserNoteData) (*models.UserNote, error)
	UpdateUserNote(ctx context.Context, userID string, noteID int, data models.UpdateUserNoteData) (*models.UserNote, error)
	DeleteUserNote(ctx context.Context, userID string, noteID int) error
	AddLinkToNote(ctx context.Context, noteID int, data models.AddLinkToNoteData) (*models.UserNoteLink, error)
	RemoveLinkFromNote(ctx context.Context, noteID int, linkID int) error
	PublishNoteToForum(ctx context.Context, userID string, noteID int, publishDetails models.ForumPostPublishDetails) (*models.ForumPost, error)

	// Notifications
	GetNotifications(ctx context.Context, userID string, page, limit int) ([]models.Notification, int, error)

	// Favorite Artworks
	GetFavArtworks(ctx context.Context, userID string, page, limit int) ([]models.UserFavArtworkEntry, int, error)

	// Saved Forum Posts
	GetSavedForumPosts(ctx context.Context, userID string, page, limit int) ([]models.UserSavedPostEntry, int, error)

	// Admin
	AdminListUsers(ctx context.Context, page, limit int) ([]models.User, int, error)
	AdminUpdateUserRole(ctx context.Context, targetUserID string, newRole string) error
}

type Service struct {
	userRepo RepositoryInterface
	// For simplicity, userNote specific methods are on RepositoryInterface for now.
	// In a larger system, userNoteRepo might be a separate RepositoryInterface.
	forumSvc          forum.ServiceInterface    // Injected for publishing notes
	emailer           emailSvc.ServiceInterface // For sending emails
	templateManager   *emailSvc.TemplateManager
	jwtSecret         string
	clientOrigin      string // For sending activation and password reset emails (domain name)
	adminEmail        string
	googleOAuthConfig *oauth2.Config
}

func NewService(
	userRepo RepositoryInterface,
	forumSvc forum.ServiceInterface,
	emailer emailSvc.ServiceInterface,
	tm *emailSvc.TemplateManager,
	JWTSecretFromConfig string,
	clientOriginFromConfig string,
	adminEmailFromConfig string,
	googleOAuthConfig *oauth2.Config,
) ServiceInterface {
	return &Service{
		userRepo:          userRepo,
		forumSvc:          forumSvc,
		emailer:           emailer,
		templateManager:   tm,
		jwtSecret:         JWTSecretFromConfig,
		clientOrigin:      clientOriginFromConfig,
		adminEmail:        adminEmailFromConfig,
		googleOAuthConfig: googleOAuthConfig,
	}
}

// A struct to unmarshal the Google user info response
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// Allows other packages (like the handler) to know the frontend URL for redirects.
func (s *Service) GetClientOrigin() string {
	return s.clientOrigin
}

func (s *Service) Signup(ctx context.Context, req models.SignupRequest) (*models.User, error) {
	// 1. Check if user with that email already exists
	_, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		// Some other database error occurred
		return nil, fmt.Errorf("service.Signup.FindByEmail: %w", err)
	}
	if err == nil {
		// User was found, email is taken
		return nil, models.ErrConflict
	}

	// 2. Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("service.Signup.HashPassword: %w", err)
	}

	// 3. Create activation token
	activationToken, err := utils.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("service.Signup.GenerateToken: %w", err)
	}
	expiresAt := time.Now().Add(time.Minute * 30)
	log.Println("GENERATED ACTIVATION TOKEN:", activationToken)

	// 4. Create the inactive user in the database
	newUser := &models.User{
		Nickname: req.Nickname,
		Email:    req.Email,
		Role:     models.RoleNormalUser, // Default role
	}
	createdUser, err := s.userRepo.CreateInactiveUser(ctx, newUser, string(hashedPassword), activationToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("service.Signup.CreateUser: %w", err)
	}

	// 5. Send activation email
	activationURL := fmt.Sprintf("%s/activate?token=%s", s.clientOrigin, activationToken, expiresAt)

	htmlContent, err := s.templateManager.GenerateActivateAccountEmailHTML(emailSvc.TemplateData{
		Name: createdUser.Nickname,
		Link: activationURL,
	})
	if err != nil {
		// Log the error but don't fail the whole signup process
		log.Printf("Failed to generate activation email HTML: %v", err)
		return createdUser, nil
	}

	emailSubject := "[Circuit] Welcome! Please Activate Your Account"
	plainTextContent := fmt.Sprintf("Thank you for signing up! Please click the following link in 30 minutes to activate your account: %s", activationURL)

	go func() {
		// Run in a goroutine so it doesn't block the user's signup response
		err := s.emailer.SendEmail(context.Background(), createdUser.Email, emailSubject, plainTextContent, htmlContent)
		if err != nil {
			log.Printf("Failed to send activation email to %s: %v", createdUser.Email, err)
		}
	}()

	return createdUser, nil
}

// private helper function to generate AuthResponse
func (s *Service) generateAuthResponse(user *models.User) (*models.AuthResponse, error) {
	// 1. Create claims for JWT
	claims := &models.JwtCustomClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 30)), // 1 month expiry
		},
	}

	// 2. Create access token with claims
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 3. Generate encoded token and send it as response
	tokenSignedString, err := accessToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	user.PasswordHash = nil

	return &models.AuthResponse{
		AccessToken: tokenSignedString,
		User:        user,
	}, nil
}

func (s *Service) ActivateUserAndLogin(ctx context.Context, token string) (*models.AuthResponse, error) {
	activatedUser, err := s.userRepo.ActivateUser(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("service.ActivateUserAndLogin: %w", err)
	}

	return s.generateAuthResponse(activatedUser)
}

func (s *Service) ResendActivationEmail(ctx context.Context, email string) error {
	// 1. Find user by email
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// If user not found, do nothing and return nil to hide existence.
		if errors.Is(err, models.ErrNotFound) {
			log.Printf("INFO: Activation resend requested for non-existent email: %s", email)
			return nil
		}
		return fmt.Errorf("service.ResendActivationEmail.FindByEmail: %w", err)
	}

	// 2. Check if user is already active
	if user.IsActive {
		log.Printf("INFO: Activation resend requested for already active user: %s", email)
		return nil // Do nothing, don't signal that they are active.
	}

	// 3. Generate a new activation token
	activationToken, err := utils.GenerateSecureToken(32)
	if err != nil {
		return fmt.Errorf("service.ResendActivationEmail.GenerateToken: %w", err)
	}
	expiresAt := time.Now().Add(time.Minute * 30)

	// 4. Update the user record with the new token
	if err := s.userRepo.UpdateActivationToken(ctx, user.ID, activationToken, expiresAt); err != nil {
		return fmt.Errorf("service.ResendActivationEmail.UpdateToken: %w", err)
	}

	// 5. Send the new activation email
	activationURL := fmt.Sprintf("%s/activate?token=%s", s.clientOrigin, activationToken)

	htmlContent, err := s.templateManager.GenerateActivateAccountEmailHTML(emailSvc.TemplateData{
		Name: user.Nickname,
		Link: activationURL,
	})
	if err != nil {
		// Log the error but don't fail the whole signup process
		log.Printf("Failed to generate re-activation email HTML: %v", err)
		return nil
	}

	emailSubject := "[Jingdezhen Ceramics] Activate Your Account (New Link)"
	plainTextContent := fmt.Sprintf("Please click the following link in 30 minutes to activate your account: %s", activationURL)

	go func() {
		// Run in a goroutine so it doesn't block the user's signup response
		err := s.emailer.SendEmail(context.Background(), email, emailSubject, plainTextContent, htmlContent)
		if err != nil {
			log.Printf("Failed to send re-activation email to %s: %v", email, err)
		}
	}()

	return nil
}

func (s *Service) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	// 1. Find user by email
	userWithHash, err := s.userRepo.FindByEmail(ctx, req.Email) // This needs to return password hash
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, models.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("service.Login.FindByEmail: %w", err)
	}

	// 2. Compare the provided password with the stored hash
	if userWithHash.PasswordHash == nil {
		// This user was created via OAuth and has no password.
		return nil, models.ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(*userWithHash.PasswordHash), []byte(req.Password))
	if err != nil {
		// Passwords don't match
		return nil, models.ErrInvalidCredentials
	}

	// 3. Check if the user is active
	if !userWithHash.IsActive {
		return nil, models.ErrInactiveAccount
	}

	// 4. Use helper function to generate JWT and AuthResponse
	return s.generateAuthResponse(userWithHash)
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	// 1. Find user by email
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// Even if user not found, return success to prevent email enumeration attacks
		log.Printf("Password reset requested for non-existent email: %s", err)
		return nil
	}

	// 2. Gnerate reset token and expiry
	token, err := utils.GenerateSecureToken(32)
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(15 * time.Minute) // token is valid for 15 minutes
	log.Println("GENERATE RESET PASSWORD TOKEN: ", token)

	// 3. Save token and expiry to user record
	if err := s.userRepo.SetPasswordResetToken(ctx, user.ID, token, expiresAt); err != nil {
		return err
	}

	// 4. Send password reset email
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.clientOrigin, token)

	htmlContent, err := s.templateManager.GenerateResetPasswordEmailHTML(emailSvc.TemplateData{
		Name: user.Nickname,
		Link: resetURL,
	})
	if err != nil {
		// Log the error but don't fail the whole signup process
		log.Printf("Failed to generate re-activation email HTML: %v", err)
		return nil
	}

	emailSubject := "[Jingdezhen Ceramics] Reset Your Password"
	plainTextContent := fmt.Sprintf("Please click the following link in 15 minutes to reset your password: %s", resetURL)

	go func() {
		// Run in a goroutine so it doesn't block the user's signup response
		err := s.emailer.SendEmail(context.Background(), email, emailSubject, plainTextContent, htmlContent)
		if err != nil {
			log.Printf("Failed to send password resetting email to %s: %v", email, err)
		}
	}()

	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token string, newPassword string) (*models.AuthResponse, error) {
	// 1. Find user by reset token and check expiry
	// Read and Security Check: verify the token matches AND has not expired
	user, err := s.userRepo.FindByPasswordResetToken(ctx, token)
	if err != nil {
		if errors.Is(err, models.ErrInvalidToken) {
			return nil, models.ErrInvalidToken // Token not found or expired
		}
		return nil, fmt.Errorf("service.ResetPassword.FindToken: %w", err)
	}

	// 2. Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 3. Update the user's password and clear the reset token
	// Write and State Change: update users table in database
	if err := s.userRepo.UpdatePasswordAndClearResetToken(ctx, user.ID, string(hashedPassword)); err != nil {
		return nil, err
	}

	// 4. Log the user in by issuing a JWT
	return s.generateAuthResponse(user)
}

// HandleGoogleLogin generates the redirect URL for the user.
func (s *Service) HandleGoogleLogin() (string, string, error) {
	// Generates the URL the user should be redirected to.
	// The state parameter is crucial for CSRF protection.
	// It should be a random, non-guessable string.
	// In a production app, you'd generate this, store it in a short-lived, secure,
	// HttpOnly cookie, and then compare it in the callback handler.
	state, err := utils.GenerateSecureToken(16)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state for google login: %w", err)
	}
	// This generates a URL like:
	// https://accounts.google.com/o/oauth2/v2/auth?client_id=...&redirect_uri=...&response_type=code&scope=...&state=...
	url := s.googleOAuthConfig.AuthCodeURL(state)
	return url, state, nil
}

// HandleGoogleCallback processes the callback from Google, completing the login/signup.
func (s *Service) HandleGoogleCallback(ctx context.Context, code string) (*models.AuthResponse, error) {
	// 1. Exchange authorization code for a token from Google
	token, err := s.googleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google code exchange failed: %w", err)
	}

	// 2. Use the token to get the user's info from Google's API.
	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info from google: %w", err)
	}
	defer response.Body.Close()

	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading user info response body: %w", err)
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(contents, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	if !userInfo.VerifiedEmail {
		return nil, fmt.Errorf("google email not verified")
	}

	// 3. Find or create user in database
	user, err := s.userRepo.FindByEmail(ctx, userInfo.Email)
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		return nil, fmt.Errorf("db error while finding user by email: %w", err)
	}

	if errors.Is(err, models.ErrNotFound) {
		// User does not exist, create them
		newUser := &models.User{
			Nickname:       userInfo.Name,
			Email:          userInfo.Email,
			AvatarURL:      &userInfo.Picture,
			Role:           models.RoleNormalUser,
			AuthProvider:   "google",
			AuthProviderID: userInfo.ID,
			IsActive:       true,
		}
		user, err = s.userRepo.CreateOAuthUser(ctx, newUser)
		if err != nil {
			return nil, err
		}
	}
	// If the user was found, you might want to check if their AuthProvider is "email"
	// and potentially link the Google account by setting AuthProvider and AuthProviderID.
	// For now, we'll just log them in.

	// 4. Issue JWT for this user.
	return s.generateAuthResponse(user)
}

func (s *Service) GetUserProfile(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		// Map repository errors to service-level errors if needed
		return nil, fmt.Errorf("service.GetUserProfile: %w", err)
	}
	return user, nil
}

func (s *Service) UpdateUserProfile(ctx context.Context, userID string, data models.UserUpdateData) (*models.User, error) {
	// Check if nickname is unique if that's a requirement (would need repo method)
	if data.Nickname != nil {
		existingUserWithNickname, err := s.userRepo.FindByNickname(ctx, *data.Nickname)
		if err != nil && !errors.Is(err, models.ErrNotFound) {
			return nil, fmt.Errorf("failed to check nickname uniqueness: %w", err)
		}
		if existingUserWithNickname != nil && existingUserWithNickname.ID != userID {
			return nil, models.ErrNicknameTaken
		}
	}

	updatedUser, err := s.userRepo.Update(ctx, userID, data)
	if err != nil {
		return nil, fmt.Errorf("service.UpdateUserProfile: %w", err)
	}
	return updatedUser, nil
}

func (s *Service) HandleContactSubmission(ctx context.Context, data models.ContactFormData) error {
	// 1. Sanitize inputs
	log.Printf("Contact Form Submitted: Name: %s, Email: %s, Subject: %s",
		data.Name, data.Email, data.Subject)

	emailSubject := fmt.Sprintf("New Contact Form Submission: %s", data.Subject)
	emailBody := fmt.Sprintf(
		"You have received a new message from the contact form:\n\nName: %s\nEmail: %s\n\nMessage:\n%s",
		data.Name, data.Email, data.Message,
	)

	// 2. Send an email to the admin using an email service
	err := s.emailer.SendEmail(ctx, s.adminEmail, emailSubject, emailBody, "")
	if err != nil {
		log.Printf("ERROR sending contact email: %v", err)
		// Decide if this should be a user-facing error or just logged
		return fmt.Errorf("failed to send contact message: %w", err)
	}
	log.Printf("SIMULATED: Email sent to %s, Subject: %s", s.adminEmail, emailSubject)

	return nil // Simulate success
}

// --- User Notes ---
func (s *Service) ListUserNotes(ctx context.Context, userID string, page, limit int) ([]models.UserNote, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	} // Default/max limit
	notes, total, err := s.userRepo.ListUserNotes(ctx, userID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListUserNotes: %w", err)
	}
	return notes, total, nil
}

func (s *Service) GetUserNoteDetails(ctx context.Context, userID string, noteID int) (*models.UserNote, error) {
	note, err := s.userRepo.GetUserNoteByID(ctx, noteID, userID) // Repo checks ownership
	if err != nil {
		return nil, fmt.Errorf("service.GetUserNoteDetails: %w", err)
	}
	links, err := s.userRepo.GetLinksForNote(ctx, noteID)
	if err != nil {
		log.Printf("Failed to get links for note %d", noteID)
		return note, models.ErrNotFound
	}
	note.Links = links
	return note, nil
}

func (s *Service) CreateUserNote(ctx context.Context, userID string, data models.CreateUserNoteData) (*models.UserNote, error) {
	// Add business logic: e.g., check if user can create notes for this entity_type/entity_id
	note, err := s.userRepo.CreateUserNote(ctx, userID, data)
	if err != nil {
		return nil, fmt.Errorf("service.CreateUserNote: %w", err)
	}
	return note, nil
}

func (s *Service) UpdateUserNote(ctx context.Context, userID string, noteID int, data models.UpdateUserNoteData) (*models.UserNote, error) {
	// userRepo.UpdateUserNote already checks ownership by including userID in query
	note, err := s.userRepo.UpdateUserNote(ctx, noteID, userID, data)
	if err != nil {
		return nil, fmt.Errorf("service.UpdateUserNote: %w", err)
	}
	return note, nil
}

func (s *Service) DeleteUserNote(ctx context.Context, userID string, noteID int) error {
	// userRepo.DeleteUserNote already checks ownership by including userID in query
	err := s.userRepo.DeleteUserNote(ctx, noteID, userID)
	if err != nil {
		return fmt.Errorf("service.DeleteUserNote: %w", err)
	}
	return nil
}

func (s *Service) AddLinkToNote(ctx context.Context, noteID int, data models.AddLinkToNoteData) (*models.UserNoteLink, error) {
	link, err := s.userRepo.AddLinkToNote(ctx, noteID, data)
	if err != nil {
		return nil, fmt.Errorf("service.AddLinkToNote: %w", err)
	}
	return link, nil
}

func (s *Service) RemoveLinkFromNote(ctx context.Context, noteID int, linkID int) error {
	err := s.userRepo.RemoveLinkFromNote(ctx, noteID, linkID)
	if err != nil {
		return fmt.Errorf("service.RemoveLinkFromNote: %w", err)
	}
	return nil
}

func (s *Service) PublishNoteToForum(ctx context.Context, userID string, noteID int, publishDetails models.ForumPostPublishDetails) (*models.ForumPost, error) {
	// 1. Start a transaction
	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("service.PublishNoteToForum.BeginTx: %w", err)
	}
	// Defer a rollback in case anything goes wrong.
	// If the function returns with an error, this will execute and cancel the transaction
	defer tx.Rollback(ctx)

	// 2. Create transaction-scoped repositories
	userRepoWithTx := s.userRepo.WithTx(tx)
	forumRepoWithTx := s.forumRepo.WithTx(tx)

	// 3. Perform operations within the transaction
	// Validate category ID
	isValidCategory, err := s.forumSvc.IsValidCategory(ctx, publishDetails.CategoryID)
	if err != nil {
		// Log error, maybe return a generic server error or a specific "validation failed"
		return nil, fmt.Errorf("failed to validate category: %w", err)
	}
	if !isValidCategory {
		return nil, models.ErrInvalidForumPostCategoryID
	}

	// Get the note using transactional user repo
	note, err := userRepoWithTx.GetUserNoteByID(ctx, noteID, userID)
	if err != nil {
		return nil, fmt.Errorf("service.PublishNoteToForum.GetNote: %w", err)
	}
	if note.IsPublishedToForum {
		// Optionally, you could return the existing forum post if note.ForumPostID is not nil
		return nil, models.ErrConflict
	}

	// Prepare data for creating forum post
	createPostData := models.CreateForumPostData{ // Assuming this struct exists in models
		Title:      publishDetails.Title,
		Content:    note.Content, // Use content from the note
		CategoryID: publishDetails.CategoryID,
		Tags:       publishDetails.Tags,
		// UserID is handled by forumService.CreatePost based on the authenticated user
	}

	// Create post usingg transactional forum repo
	createdPost, err := forumRepoWithTx.CreatePost(ctx, userID, createPostData) // userID passed here is the authenticated user
	if err != nil {
		return nil, fmt.Errorf("service.PublishNoteToForum.CreatePost: %w", err)
	}

	// Mark note as published
	err = userRepoWithTx.MarkNoteAsPublished(ctx, noteID, createdPost.ID)
	if err != nil {
		// Because we are in a transaction, if this fails, the defer tx.Rollback(ctx)
		// will automatically undo the forum post creation.
		return nil, fmt.Errorf("service.PublishNoteToForum.MarkNoteAsPublished: %w", err)
	}

	// --- Step 4: Commit the Transaction ---
	// If we reach this point, all operations were successful.
	// We commit the transaction to make all changes permanent.
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("service.PublishNoteToForum.Commit: %w", err)
	}

	return createdPost, nil
}

func (s *Service) GetNotifications(ctx context.Context, userID string, page, limit int) ([]models.Notification, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	} // Default/max limit
	notifications, total, err := s.userRepo.GetNotifications(ctx, userID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("service.GetNotifications: %w", err)
	}
	return notifications, total, nil
}

func (s *Service) GetFavArtworks(ctx context.Context, userID string, page, limit int) ([]models.UserFavArtworkEntry, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	} // Default/max limit
	favArtworks, total, err := s.userRepo.GetFavArtworks(ctx, userID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("service.GetFavArtworks: %w", err)
	}
	return favArtworks, total, nil
}

func (s *Service) GetSavedForumPosts(ctx context.Context, userID string, page, limit int) ([]models.UserSavedPostEntry, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	} // Default/max limit
	savedForumPosts, total, err := s.userRepo.GetSavedForumPosts(ctx, userID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("service.GetSavedForumPosts: %w", err)
	}
	return savedForumPosts, total, nil
}

// --- Admin Service Methods ---
func (s *Service) AdminListUsers(ctx context.Context, page, limit int) ([]models.User, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.userRepo.ListAll(ctx, page, limit)
}

func (s *Service) AdminUpdateUserRole(ctx context.Context, targetUserID string, newRole string) error {
	// Add validation for newRole if it's not a predefined valid role
	if newRole != models.RoleAdmin && newRole != models.RoleNormalUser {
		return fmt.Errorf("service.AdminUpdateUserRole: invalid role '%s'", newRole)
	}
	// Check if targetUserID exists
	_, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("service.AdminUpdateUserRole: target user not found: %w", err)
	}

	return s.userRepo.UpdateRole(ctx, targetUserID, newRole)
}
