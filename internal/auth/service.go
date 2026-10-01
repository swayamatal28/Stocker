package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type SessionStore interface {
	CreateRefreshSession(context.Context, string, []byte, time.Time) error
	RotateRefreshSession(context.Context, []byte, []byte, time.Time) (string, error)
	RevokeRefreshSession(context.Context, []byte) error
	UserRole(context.Context, string) (string, error)
}

type Service struct {
	secret                []byte
	accessTTL, refreshTTL time.Duration
	sessions              SessionStore
}

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func New(secret string, accessTTL, refreshTTL time.Duration, sessions SessionStore) *Service {
	return &Service{[]byte(secret), accessTTL, refreshTTL, sessions}
}

func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", errors.New("password must contain at least 12 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(b), err
}
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (s *Service) NewPair(ctx context.Context, userID, role string) (string, string, time.Time, error) {
	access, err := s.access(userID, role)
	if err != nil {
		return "", "", time.Time{}, err
	}
	refresh, hash, err := newRefresh()
	if err != nil {
		return "", "", time.Time{}, err
	}
	expires := time.Now().UTC().Add(s.refreshTTL)
	if err := s.sessions.CreateRefreshSession(ctx, userID, hash, expires); err != nil {
		return "", "", time.Time{}, err
	}
	return access, refresh, expires, nil
}

func (s *Service) Rotate(ctx context.Context, old string) (string, string, time.Time, error) {
	oldHash := sha256.Sum256([]byte(old))
	fresh, freshHash, err := newRefresh()
	if err != nil {
		return "", "", time.Time{}, err
	}
	expires := time.Now().UTC().Add(s.refreshTTL)
	uid, err := s.sessions.RotateRefreshSession(ctx, oldHash[:], freshHash, expires)
	if err != nil {
		return "", "", time.Time{}, err
	}
	role, err := s.sessions.UserRole(ctx, uid)
	if err != nil {
		return "", "", time.Time{}, err
	}
	access, err := s.access(uid, role)
	return access, fresh, expires, err
}
func (s *Service) Revoke(ctx context.Context, token string) error {
	h := sha256.Sum256([]byte(token))
	return s.sessions.RevokeRefreshSession(ctx, h[:])
}

func (s *Service) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	}, jwt.WithIssuer("stocker-api"), jwt.WithExpirationRequired())
	if err != nil || !parsed.Valid {
		return nil, errors.New("invalid access token")
	}
	return claims, nil
}

func (s *Service) access(uid, role string) (string, error) {
	now := time.Now().UTC()
	claims := Claims{Role: role, RegisteredClaims: jwt.RegisteredClaims{Subject: uid, Issuer: "stocker-api", ID: uuid.NewString(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL))}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}
func newRefresh() (string, []byte, error) {
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(token))
	return token, h[:], nil
}
