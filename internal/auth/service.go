package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	db "github.com/Nysonn/unibuzz-api/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	store  *db.Queries
	redis  *redis.Client
	tokens *TokenManager
}

func NewService(store *db.Queries, redis *redis.Client, tokens *TokenManager) *Service {
	return &Service{
		store:  store,
		redis:  redis,
		tokens: tokens,
	}
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *Service) Register(ctx context.Context, input db.CreateUserParams) (*db.User, error) {

	hash, err := HashPassword(input.PasswordHash)
	if err != nil {
		return nil, err
	}

	input.PasswordHash = hash

	user, err := s.store.CreateUser(ctx, input)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Service) Login(ctx context.Context, email string, password string) (string, string, error) {

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return "", "", err
	}

	err = CheckPassword(password, user.PasswordHash)
	if err != nil {
		return "", "", err
	}

	role := "user"
	if user.IsAdmin.Bool {
		role = "admin"
	}

	accessToken, err := s.tokens.GenerateAccessToken(uuid.UUID(user.ID.Bytes), role)
	if err != nil {
		return "", "", err
	}

	refreshToken := uuid.New().String()
	refreshHash := hashToken(refreshToken)

	session, err := s.store.CreateSession(ctx, db.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: refreshHash,
		ExpiresAt:        pgtype.Timestamp{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})

	if err != nil {
		return "", "", err
	}

	s.redis.Set(ctx, "session:"+uuid.UUID(session.ID.Bytes).String(), refreshHash, 7*24*time.Hour)

	return accessToken, refreshToken, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (string, string, error) {

	hash := hashToken(refreshToken)

	session, err := s.store.GetSessionByToken(ctx, hash)
	if err != nil {
		return "", "", err
	}

	err = s.store.RevokeSession(ctx, session.ID)
	if err != nil {
		return "", "", err
	}

	user, err := s.store.GetUserByID(ctx, session.UserID)
	if err != nil {
		return "", "", err
	}

	role := "user"
	if user.IsAdmin.Bool {
		role = "admin"
	}

	accessToken, err := s.tokens.GenerateAccessToken(uuid.UUID(session.UserID.Bytes), role)
	if err != nil {
		return "", "", err
	}

	newRefresh := uuid.New().String()

	newHash := hashToken(newRefresh)

	_, err = s.store.CreateSession(ctx, db.CreateSessionParams{
		UserID:           session.UserID,
		RefreshTokenHash: newHash,
		ExpiresAt:        pgtype.Timestamp{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})

	if err != nil {
		return "", "", err
	}

	return accessToken, newRefresh, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {

	hash := hashToken(refreshToken)

	session, err := s.store.GetSessionByToken(ctx, hash)
	if err != nil {
		return err
	}

	return s.store.RevokeSession(ctx, session.ID)
}
