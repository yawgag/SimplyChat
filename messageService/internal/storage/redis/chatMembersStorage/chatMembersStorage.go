package chatMembersStorage

//
// this redis storage exist, to reduce the number of database queries
// it store all the users who are in the chat
//

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrorListDoesntExist error         = errors.New("list doesn't exist")
	MembersListTTL       time.Duration = 5 * time.Minute
)

type chatMemberStorage struct {
	client *redis.Client
}

type Handler interface {
	CreateMembersList(ctx context.Context, chatId int, usersLogin ...string) error
	GetMembersList(ctx context.Context, chatId int) ([]string, error)
	AddMemberToList(ctx context.Context, chatId int, userLogin string) error
	DeleteMemberFromList(ctx context.Context, chatId int, userLogin string) error

	UpdateListTTL(ctx context.Context, chatId int) error
}

func (c *chatMemberStorage) CreateMembersList(ctx context.Context, chatId int, usersLogin ...string) error {
	err := c.client.SAdd(ctx, chatKey(chatId), usersLogin).Err()
	if err != nil {
		return fmt.Errorf("[CreateMembersList] error: %s", err)
	}
	err = c.UpdateListTTL(ctx, chatId)
	if err != nil {
		return fmt.Errorf("[CreateMembersList] error: %s", err)
	}
	return nil
}

func (c *chatMemberStorage) GetMembersList(ctx context.Context, chatId int) ([]string, error) {
	members, err := c.client.SMembers(ctx, chatKey(chatId)).Result()
	if err != nil {
		return nil, fmt.Errorf("[GetMembersList] error: %s", err)
	}

	err = c.UpdateListTTL(ctx, chatId)
	if err != nil {
		return nil, fmt.Errorf("[CreateMembersList] error: %s", err)
	}
	return members, nil
}

func (c *chatMemberStorage) AddMemberToList(ctx context.Context, chatId int, userLogin string) error {
	err := c.client.SAdd(ctx, chatKey(chatId), userLogin).Err()
	if err != nil {
		return fmt.Errorf("[AddMemberToList] error: %s", err)
	}
	err = c.UpdateListTTL(ctx, chatId)
	if err != nil {
		return fmt.Errorf("[AddMemberToList] error: %s", err)
	}
	return nil
}

func (c *chatMemberStorage) DeleteMemberFromList(ctx context.Context, chatId int, userLogin string) error {
	err := c.client.SRem(ctx, chatKey(chatId), userLogin).Err()
	if err != nil {
		return fmt.Errorf("[DeleteMemberFromList] error: %s", err)
	}
	err = c.UpdateListTTL(ctx, chatId)
	if err != nil {
		return fmt.Errorf("[DeleteMemberFromList] error: %s", err)
	}
	return nil
}

func (c *chatMemberStorage) UpdateListTTL(ctx context.Context, chatId int) error {
	err := c.client.Expire(ctx, chatKey(chatId), MembersListTTL).Err()
	if err != nil {
		return fmt.Errorf("[UpdateListTTL] error: %s", err)
	}
	return nil
}

func NewChatMemberStorage(ctx context.Context, redisAddr string) (Handler, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("[NewChatUsersStorage] error: %s", err)
	}

	out := &chatMemberStorage{
		client: redisClient,
	}

	return out, nil
}

func chatKey(chatId int) string {
	return "chat:" + strconv.Itoa(chatId)
}
