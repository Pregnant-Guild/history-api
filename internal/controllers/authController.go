package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/models"
	"history-api/internal/services"
	"history-api/pkg/validator"
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
			Status:  false,
			Message: err.Error(),
		})
	}

	res, err := h.service.Signin(ctx, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    res.AccessToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    res.RefreshToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
	})

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
			Status:  false,
			Message: err.Error(),
		})
	}

	res, err := h.service.Signup(ctx, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    res.AccessToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    res.RefreshToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
	})

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
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

	res, err := h.service.RefreshToken(ctx, c.Locals("uid").(string))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    res.AccessToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    res.RefreshToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
	})

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
			Status:  false,
			Message: err.Error(),
		})
	}

	res, err := h.service.VerifyToken(ctx, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
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
			Status:  false,
			Message: err.Error(),
		})
	}

	err := h.service.CreateToken(ctx, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
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
			Status:  false,
			Message: err.Error(),
		})
	}

	err := h.service.ForgotPassword(ctx, dto)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
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

	redirect := c.Query("redirect")
	if redirect == "" {
		redirect = "http://localhost:3000"
	}

	data := models.OAuthState{
		State:       state,
		RedirectURL: redirect,
	}

	b, _ := json.Marshal(data)
	encoded := base64.URLEncoding.EncodeToString(b)

	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
	})

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

	c.ClearCookie("oauth_state")

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

	res, err := h.service.SigninWithGoogle(ctx, &googleUser)
	if err != nil {
		return c.Status(500).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    res.AccessToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    res.RefreshToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
	})

	allowed := map[string]bool{
		"http://localhost:3000":  true,
		"https://localhost:3000": true,
		"http://localhost:3001":  true,
		"https://localhost:3001": true,
		"http://localhost:5500":  true,
	}

	redirectURL := data.RedirectURL
	if !allowed[redirectURL] {
		redirectURL = "http://localhost:3000"
	}

	return c.Redirect().To(redirectURL)
}

func (h *AuthController) Logout(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	userId := c.Locals("uid").(string)

	err := h.service.Logout(ctx, userId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   true,
		Path:     "/",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   true,
		Path:     "/",
	})

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status:  true,
		Message: "Logged out successfully",
	})
}
