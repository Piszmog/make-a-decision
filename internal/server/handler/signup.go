package handler

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"unicode"

	"github.com/Piszmog/make-a-decision/internal/components/auth"
	"github.com/Piszmog/make-a-decision/internal/db/queries"
	"github.com/Piszmog/make-a-decision/internal/server/utils"
	"golang.org/x/crypto/bcrypt"
)

// Password validation constants
const (
	minPasswordLength = 8
	bcryptCost        = 12 // Recommended cost for bcrypt
)

// passwordRequirements validates password strength
type passwordRequirements struct {
	hasMinLength bool
	hasUpper     bool
	hasLower     bool
	hasNumber    bool
	hasSpecial   bool
}

// SignupPage renders the signup form
func (h *Handler) SignupPage(w http.ResponseWriter, r *http.Request) {
	h.html(r.Context(), w, http.StatusOK, auth.SignupPage())
}

// SignupSubmit handles the signup form submission
func (h *Handler) SignupSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse form values
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate email
	if err := validateEmail(email); err != nil {
		h.renderSignupError(ctx, w, email, err.Error())
		return
	}

	// Validate passwords match
	if password != confirmPassword {
		h.renderSignupError(ctx, w, email, "Passwords do not match")
		return
	}

	// Validate password strength
	if err := validatePassword(password); err != nil {
		h.renderSignupError(ctx, w, email, err.Error())
		return
	}

	// Check if user already exists
	exists, err := h.Database.Queries().UserExists(ctx, email)
	if err != nil {
		h.Logger.Error("Failed to check if user exists", "error", err, "email", email)
		h.renderSignupError(ctx, w, email, "An error occurred. Please try again.")
		return
	}
	if exists {
		h.renderSignupError(ctx, w, email, "An account with this email already exists")
		return
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		h.Logger.Error("Failed to hash password", "error", err)
		h.renderSignupError(ctx, w, email, "An error occurred. Please try again.")
		return
	}

	// Create user
	user, err := h.Database.Queries().CreateUser(ctx, queries.CreateUserParams{
		Email:        email,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		h.Logger.Error("Failed to create user", "error", err, "email", email)
		h.renderSignupError(ctx, w, email, "An error occurred. Please try again.")
		return
	}

	h.Logger.Info("User created successfully", "user_id", user.ID, "email", user.Email)

	// Create session automatically to sign the user in
	token, expiresAt, err := h.newSession(ctx, user.ID, r.UserAgent(), "", utils.GetClientIP(r))
	if err != nil {
		h.Logger.Error("Failed to create session after signup", "error", err)
		h.renderSignupError(ctx, w, email, "Account created but sign-in failed. Please sign in manually.")
		return
	}

	utils.SetSessionCookie(w, token, expiresAt)

	// Trigger local storage sync on frontend
	w.Header().Set("HX-Trigger", `{"syncLocalStorage": true}`)

	// Redirect to home page
	w.Header().Set("HX-Redirect", "/")
}

// renderSignupError renders the signup form with an error message
func (h *Handler) renderSignupError(ctx context.Context, w http.ResponseWriter, email, errorMsg string) {
	h.html(ctx, w, http.StatusOK, auth.SignupFormFields(email, errorMsg))
}

// validateEmail validates email format using Go's mail.ParseAddress
func validateEmail(email string) error {
	if email == "" {
		return errors.New("Email is required")
	}

	// Use Go's built-in email parser
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return errors.New("Invalid email format")
	}

	// Additional validation: ensure it has @ and domain
	if !strings.Contains(addr.Address, "@") || !strings.Contains(addr.Address, ".") {
		return errors.New("Invalid email format")
	}

	return nil
}

// validatePassword validates password strength
func validatePassword(password string) error {
	if password == "" {
		return errors.New("Password is required")
	}

	reqs := checkPasswordRequirements(password)

	if !reqs.hasMinLength {
		return errors.New("Password must be at least 8 characters long")
	}

	if !reqs.hasUpper {
		return errors.New("Password must contain at least one uppercase letter")
	}

	if !reqs.hasLower {
		return errors.New("Password must contain at least one lowercase letter")
	}

	if !reqs.hasNumber {
		return errors.New("Password must contain at least one number")
	}

	if !reqs.hasSpecial {
		return errors.New("Password must contain at least one special character")
	}

	return nil
}

// checkPasswordRequirements checks which password requirements are met
func checkPasswordRequirements(password string) passwordRequirements {
	reqs := passwordRequirements{
		hasMinLength: len(password) >= minPasswordLength,
	}

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			reqs.hasUpper = true
		case unicode.IsLower(char):
			reqs.hasLower = true
		case unicode.IsNumber(char):
			reqs.hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			reqs.hasSpecial = true
		}
	}

	return reqs
}
