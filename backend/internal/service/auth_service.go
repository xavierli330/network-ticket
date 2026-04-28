package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/xavierli/network-ticket/internal/config"
	"github.com/xavierli/network-ticket/internal/repository"
)

// Claims represents JWT claims for the application.
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// AuthService handles user authentication and JWT token management.
type AuthService struct {
	userRepo   *repository.UserRepo
	jwtConfig  *config.JWTConfig
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo *repository.UserRepo, jwtConfig *config.JWTConfig) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtConfig: jwtConfig,
	}
}

// Login authenticates a user and returns a JWT token.
func (s *AuthService) Login(ctx context.Context, username, password string) (string, *Claims, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return "", nil, fmt.Errorf("user not found: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errors.New("invalid password")
	}

	expireHours := s.jwtConfig.ExpireHours
	if expireHours <= 0 {
		expireHours = 24
	}

	now := time.Now()
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.jwtConfig.Secret))
	if err != nil {
		return "", nil, fmt.Errorf("sign token: %w", err)
	}

	return tokenStr, claims, nil
}

// ParseToken validates a JWT token and returns the claims.
func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtConfig.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
