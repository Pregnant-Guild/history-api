package controllers

import (
	"context"
	"encoding/base64"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/models"
	"history-api/internal/services"
	"history-api/pkg/config"
	"history-api/pkg/constants"
	json "history-api/pkg/jsonx"
	"history-api/pkg/validator"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

type AuthController struct {
	service services.AuthService
	oauth   *oauth2.Config
}

func NewAuthController(svc services.AuthService, oauth *oauth2.Config) *AuthController {
	return &AuthController{service: svc, oauth: oauth}
}

func authCookieSecure() bool {
	return config.GetBoolConfigWithDefault("COOKIE_SECURE", true)
}

func authCookieDomain() string {
	return config.GetConfigWithDefault("COOKIE_DOMAIN", "")
}

func authCookieSameSite() string {
	if authCookieSecure() {
		return "None"
	}
	return "Lax"
}

func setAuthCookie(c fiber.Ctx, name string, value string, duration time.Duration) {
	cookie := &fiber.Cookie{
		Name:     name,
		Value:    value,
		Expires:  time.Now().Add(duration),
		MaxAge:   int(duration.Seconds()),
		HTTPOnly: true,
		Secure:   authCookieSecure(),
		SameSite: authCookieSameSite(),
		Path:     "/",
	}
	if domain := authCookieDomain(); domain != "" {
		cookie.Domain = domain
	}
	c.Cookie(cookie)
}

func clearAuthCookie(c fiber.Ctx, name string) {
	cookie := &fiber.Cookie{
		Name:     name,
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   authCookieSecure(),
		SameSite: authCookieSameSite(),
		Path:     "/",
	}
	if domain := authCookieDomain(); domain != "" {
		cookie.Domain = domain
	}
	c.Cookie(cookie)
}

func setAuthCookies(c fiber.Ctx, res *response.AuthResponse) {
	setAuthCookie(c, "access_token", res.AccessToken, constants.AccessTokenDuration)
	setAuthCookie(c, "refresh_token", res.RefreshToken, constants.RefreshTokenDuration)
}

// Signin godoc
// @Summary Sign in a user
// @Description Authenticate user credentials and return access/refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body request.SignInDto true "Sign In credentials"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 401 {object} response.CommonResponse "Invalid credentials"
// @Failure 500 {object} response.CommonResponse
// @Router /auth/signin [post]
func (h *AuthController) Signin(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.SignInDto{}

	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := h.service.Signin(ctx, dto)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	setAuthCookies(c, res)

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

// Signup godoc
// @Summary Register a new user
// @Description Create a new user account in the system
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body request.SignUpDto true "Sign Up details"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /auth/signup [post]
func (h *AuthController) Signup(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.SignUpDto{}

	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := h.service.Signup(ctx, dto)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	setAuthCookies(c, res)

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

func (h *AuthController) getRefreshToken(c fiber.Ctx) string {
	auth := c.Get("Authorization")
	if auth != "" {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	return c.Cookies("refresh_token")
}

// RefreshToken godoc
// @Summary Refresh session tokens
// @Description Generate a new access token using a valid refresh token from context
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.CommonResponse
// @Failure 401 {object} response.CommonResponse "Unauthorized or expired refresh token"
// @Failure 500 {object} response.CommonResponse
// @Router /auth/refresh [post]
func (h *AuthController) RefreshToken(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tokenJwt := h.getRefreshToken(c)
	if tokenJwt == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Missing refresh token",
		})
	}

	res, err := h.service.RefreshToken(ctx, c.Locals("uid").(string), tokenJwt)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	setAuthCookies(c, res)

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

// VerifyToken godoc
// @Summary Verify a security token
// @Description Validate an OTP or email verification token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body request.VerifyTokenDto true "Token verification data"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /auth/token/verify [post]
func (h *AuthController) VerifyToken(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.VerifyTokenDto{}

	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := h.service.VerifyToken(ctx, dto)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

// CreateToken godoc
// @Summary Generate a new verification token
// @Description Request a new token for specific actions like email confirmation
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body request.CreateTokenDto true "Token creation request"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /auth/token/create [post]
func (h *AuthController) CreateToken(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.CreateTokenDto{}

	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	err := h.service.CreateToken(ctx, dto)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "If this email exists, an OTP has been sent",
	})
}

// ForgotPassword godoc
// @Summary Handle forgotten password
// @Description Initiate password recovery process for a user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body request.ForgotPasswordDto true "Forgot Password request"
// @Success 200 {object} response.CommonResponse
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /auth/forgot-password [post]
func (h *AuthController) ForgotPassword(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dto := &request.ForgotPasswordDto{}

	if err := validator.ValidateBodyDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	err := h.service.ForgotPassword(ctx, dto)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Data:    nil,
		Message: "Password reset successfully",
	})
}

// GoogleLogin godoc
// @Summary Initiate Google OAuth2 login
// @Description Generates a state string, sets it in a cookie, and redirects the user to Google's consent page.
// @Tags Auth
// @Success 302 {string} string "Redirect to Google"
// @Router /auth/google/login [get]
func (h *AuthController) GoogleLogin(c fiber.Ctx) error {
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	state := uuid.New().String()

	feUrl := config.GetConfigWithDefault("FRONTEND_URL", "http://localhost:3000")
	redirect := c.Query("redirect")
	if redirect == "" {
		redirect = feUrl
	}

	data := models.OAuthState{
		State:       state,
		RedirectURL: redirect,
	}

	b, _ := json.Marshal(data)
	encoded := base64.URLEncoding.EncodeToString(b)

	oauthCookie := &fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Expires:  time.Now().Add(15 * time.Minute),
		MaxAge:   int((15 * time.Minute).Seconds()),
		HTTPOnly: true,
		Secure:   authCookieSecure(),
		SameSite: authCookieSameSite(),
		Path:     "/",
	}
	if domain := authCookieDomain(); domain != "" {
		oauthCookie.Domain = domain
	}
	c.Cookie(oauthCookie)

	url := h.oauth.AuthCodeURL(encoded)
	return c.Redirect().To(url)
}

// GoogleCallback godoc
// @Summary Handle Google OAuth2 callback
// @Description Receives the auth code from Google, exchanges it for tokens, creates/logs in the user, and redirects back to the frontend with application tokens.
// @Tags Auth
// @Param state query string true "Security state string"
// @Param code query string true "Authorization code from Google"
// @Success 302 {string} string "Redirect to Frontend with JWTs"
// @Failure 401 {object} response.CommonResponse "Invalid state"
// @Failure 500 {object} response.CommonResponse "Internal Server Error"
// @Router /auth/google/callback [get]
func (h *AuthController) GoogleCallback(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	encoded := c.Query("state")

	b, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid state"})
	}

	var data models.OAuthState
	if err := json.Unmarshal(b, &data); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid state"})
	}

	stateFromCookie := c.Cookies("oauth_state")
	if data.State != stateFromCookie {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid state"})
	}

	clearAuthCookie(c, "oauth_state")

	code := c.Query("code")

	token, err := h.oauth.Exchange(ctx, code)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Token exchange failed"})
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "No id_token"})
	}

	payload, err := idtoken.Validate(ctx, idToken, h.oauth.ClientID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Token verification failed"})
	}

	googleUser := request.SigninWithGoogleDto{
		Sub:     payload.Subject,
		Email:   payload.Claims["email"].(string),
		Name:    payload.Claims["name"].(string),
		Picture: payload.Claims["picture"].(string),
	}

	res, err2 := h.service.SigninWithGoogle(ctx, &googleUser)
	if err2 != nil {
		return c.Status(err2.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err2.Message,
		})
	}

	setAuthCookies(c, res)

	allowed := map[string]bool{
		"http://localhost:3000":       true,
		"http://localhost:3001":       true,
		"https://admin.uhm.io.vn":     true,
		"https://api.uhm.io.vn":       true,
		"https://cdn.uhm.io.vn":       true,
		"https://uhm.io.vn":           true,
		"https://www.admin.uhm.io.vn": true,
		"https://www.api.uhm.io.vn":   true,
		"https://www.cdn.uhm.io.vn":   true,
		"https://www.uhm.io.vn":       true,
	}
	feUrl := config.GetConfigWithDefault("FRONTEND_URL", "http://localhost:3000")
	redirectURL := data.RedirectURL
	if !allowed[redirectURL] {
		redirectURL = feUrl
	}

	return c.Redirect().To(redirectURL)
}

// Logout godoc
// @Summary Logout user
// @Description Logout current user and revoke tokens
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.CommonResponse
// @Failure 401 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /auth/logout [post]
func (h *AuthController) Logout(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	userId := c.Locals("uid").(string)

	err := h.service.Logout(ctx, userId)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	clearAuthCookie(c, "access_token")
	clearAuthCookie(c, "refresh_token")

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "Logged out successfully",
	})
}
