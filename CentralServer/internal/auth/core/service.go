package core

import (
	"context"
	"errors"
	"time"

	"central_server/internal/auth/domain"
	"central_server/internal/auth/interfaces"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"
)

type AuthService struct {
	userRepo       interfaces.UserRepository
	jwtSecret      []byte
	googleClientID string
}

func NewAuthService(repo interfaces.UserRepository, secret string, googleClientID string) *AuthService {
	return &AuthService{
		userRepo:       repo,
		jwtSecret:      []byte(secret),
		googleClientID: googleClientID,
	}
}

func (s *AuthService) Register(ctx context.Context, username, password string) error {
	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &domain.User{
		Username: username,
		Password: string(hash),
	}

	return s.userRepo.Create(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	// Generate JWT
	claims := domain.Claims{
		Username: user.Username,
		UserID:   user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "central-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) GoogleLogin(ctx context.Context, idToken string) (string, string, error) {
	// Audience validation
	payload, err := idtoken.Validate(ctx, idToken, s.googleClientID)
	if err != nil {
		return "", "", errors.New("invalid google token: " + err.Error())
	}

	email, ok := payload.Claims["email"].(string)
	if !ok {
		return "", "", errors.New("email not found in google token")
	}

	// Check if user exists
	user, err := s.userRepo.GetByUsername(ctx, email)
	if err != nil {
		// User doesn't exist, create it
		user = &domain.User{
			Username: email,
			Password: "google-auth-no-password-" + email,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return "", "", err
		}
		user, _ = s.userRepo.GetByUsername(ctx, email)
	}

	// Generate JWT
	claims := domain.Claims{
		Username: user.Username,
		UserID:   user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "central-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}
	return tokenStr, email, nil
}

func (s *AuthService) VerifyToken(ctx context.Context, tokenString string) (*domain.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &domain.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*domain.Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
