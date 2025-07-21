package statusStorage

import (
	"context"
	"fmt"
	"messageService/internal/models"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type statusStorage struct {
	client *redis.Client
}

type Handler interface {
	GetStatus(ctx context.Context, uuid uuid.UUID) (*models.Session, error)
	SetOnline(ctx context.Context, uuid uuid.UUID) error
	SetOffline(ctx context.Context, uuid uuid.UUID) error
	SetNewActiveChat(ctx context.Context, uuid uuid.UUID, chatId int) error
	UpdateTTL(ctx context.Context, uuid uuid.UUID) error
}

const (
	StatusOnline  string        = "Online"
	StatusOffline string        = "Offline"
	onlineTTL     time.Duration = 5 * time.Minute
)

func NewStatusStorage(ctx context.Context, redisAddr string) (Handler, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("[NewStatusStorage] error: %s", err)
	}

	out := &statusStorage{
		client: redisClient,
	}

	return out, nil
}

// if user is online, return active chat.
func (s *statusStorage) GetStatus(ctx context.Context, uuid uuid.UUID) (*models.Session, error) {
	info, err := s.client.HGetAll(ctx, userKey(uuid)).Result()

	if err == redis.Nil {
		out := &models.Session{
			UserStatus: StatusOffline,
			ActiveChat: -1,
		}
		return out, nil
	}
	if err != nil {
		return nil, fmt.Errorf("[GetStatus] error: %s", err)
	}

	activeChat, err := strconv.Atoi(info["active_chat"])
	if err != nil {
		return nil, fmt.Errorf("[GetStatus] error: %s", err)
	}

	out := &models.Session{
		UserStatus: StatusOnline,
		ActiveChat: activeChat,
	}
	return out, nil
}

// change user status to online.
func (s *statusStorage) SetOnline(ctx context.Context, uuid uuid.UUID) error {
	if err := s.client.HSet(ctx, userKey(uuid), "status", StatusOnline); err != nil {
		return fmt.Errorf("[SetOnline] error: %s", err)
	}

	if err := s.client.Expire(ctx, userKey(uuid), onlineTTL).Err(); err != nil {
		return fmt.Errorf("[SetOnline] error: %s", err)
	}
	return nil
}

// change status to offline(delete user from redis)
func (s *statusStorage) SetOffline(ctx context.Context, uuid uuid.UUID) error {
	_, err := s.client.Del(ctx, userKey(uuid)).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("[SetOffline] error: %s", err)
	}
	return nil
}

func (s *statusStorage) SetNewActiveChat(ctx context.Context, uuid uuid.UUID, chatId int) error {

	key := userKey(uuid)

	status, err := s.client.HGet(ctx, key, "status").Result()
	if err != nil {
		return fmt.Errorf("[SetNewActiveChat] error: %s", err)
	}

	if status != StatusOnline {
		return fmt.Errorf("[SetNewActiveChat] error: can't set active chat for offline user")
	}

	if err := s.client.HSet(ctx, key, "active_chat", chatId).Err(); err != nil {
		return fmt.Errorf("[SetNewActiveChat] error: %s", err)
	}
	if err := s.client.Expire(ctx, key, onlineTTL).Err(); err != nil {
		return fmt.Errorf("[SetNewActiveChat] error: %s", err)
	}
	return nil
}

func (s *statusStorage) UpdateTTL(ctx context.Context, uuid uuid.UUID) error {
	if err := s.client.Expire(ctx, userKey(uuid), onlineTTL).Err(); err != nil {
		return fmt.Errorf("[UpdateTTL] error: %s", err)
	}
	return nil
}

func userKey(uuid uuid.UUID) string {
	return "user:" + uuid.String()
}
