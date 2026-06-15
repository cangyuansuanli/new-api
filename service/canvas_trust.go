package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const canvasTrustRedisPrefix = "canvas_trust:"

var (
	canvasTrustMemoryStore sync.Map
	canvasTrustCleanupOnce sync.Once
)

type canvasTrustMemoryEntry struct {
	UserID    int
	ExpiresAt time.Time
}

type CanvasTrustUser struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	IsAdmin     bool   `json:"is_admin"`
}

var (
	ErrCanvasTrustDisabled = errors.New("canvas trust is not configured")
	ErrCanvasTrustInvalid  = errors.New("invalid or expired canvas trust token")
	ErrCanvasTrustUser     = errors.New("invalid canvas trust user")
)

type CanvasTrustUserProfile struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	Group      string `json:"group"`
	Quota      int    `json:"quota"`
	UsedQuota  int    `json:"used_quota"`
	ServerAddr string `json:"server_address"`
}

type CanvasTrustTokenItem struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Key                string `json:"key"`
	Status             int    `json:"status"`
	RemainQuota        int    `json:"remain_quota"`
	UnlimitedQuota     bool   `json:"unlimited_quota"`
	ModelLimitsEnabled bool   `json:"model_limits_enabled"`
	ModelLimits        string `json:"model_limits"`
	Group              string `json:"group"`
}

func canvasTrustTTL() time.Duration {
	return time.Duration(setting.CanvasTrustTokenTTL) * time.Second
}

func startCanvasTrustCleanup() {
	canvasTrustCleanupOnce.Do(func() {
		go func() {
			for {
				time.Sleep(5 * time.Minute)
				now := time.Now()
				canvasTrustMemoryStore.Range(func(key, value any) bool {
					entry, ok := value.(canvasTrustMemoryEntry)
					if !ok || !entry.ExpiresAt.After(now) {
						canvasTrustMemoryStore.Delete(key)
					}
					return true
				})
			}
		}()
	})
}

func CreateCanvasTrustToken(userID int) (string, error) {
	if !setting.CanvasTrustConfigured() {
		return "", ErrCanvasTrustDisabled
	}
	if userID <= 0 {
		return "", errors.New("invalid user id")
	}

	token, err := randomCanvasTrustToken()
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(map[string]int{"user_id": userID})
	if err != nil {
		return "", err
	}

	key := canvasTrustRedisPrefix + token
	ttl := canvasTrustTTL()
	if common.RedisEnabled {
		if err := common.RedisSet(key, string(payload), ttl); err != nil {
			return "", err
		}
		return token, nil
	}

	startCanvasTrustCleanup()
	canvasTrustMemoryStore.Store(key, canvasTrustMemoryEntry{
		UserID:    userID,
		ExpiresAt: time.Now().Add(ttl),
	})
	return token, nil
}

func VerifyCanvasTrustToken(token string) (*CanvasTrustUser, error) {
	if !setting.CanvasTrustConfigured() {
		return nil, ErrCanvasTrustDisabled
	}
	token = trimCanvasTrustToken(token)
	if token == "" {
		return nil, ErrCanvasTrustInvalid
	}

	userID, err := consumeCanvasTrustToken(token)
	if err != nil {
		return nil, err
	}

	user, err := model.GetUserById(userID, false)
	if err != nil || user == nil || user.Id <= 0 {
		return nil, ErrCanvasTrustInvalid
	}
	if user.Status != common.UserStatusEnabled {
		return nil, errors.New("user is disabled")
	}

	return &CanvasTrustUser{
		ID:          user.Id,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Email:       user.Email,
		IsAdmin:     user.Role >= common.RoleAdminUser,
	}, nil
}

func consumeCanvasTrustToken(token string) (int, error) {
	key := canvasTrustRedisPrefix + token
	if common.RedisEnabled {
		raw, err := common.RedisGet(key)
		if err != nil || raw == "" {
			return 0, ErrCanvasTrustInvalid
		}
		_ = common.RedisDel(key)

		var payload struct {
			UserID int `json:"user_id"`
		}
		if err := json.Unmarshal([]byte(raw), &payload); err != nil || payload.UserID <= 0 {
			return 0, ErrCanvasTrustInvalid
		}
		return payload.UserID, nil
	}

	startCanvasTrustCleanup()
	value, ok := canvasTrustMemoryStore.LoadAndDelete(key)
	if !ok {
		return 0, ErrCanvasTrustInvalid
	}
	entry, ok := value.(canvasTrustMemoryEntry)
	if !ok || !entry.ExpiresAt.After(time.Now()) || entry.UserID <= 0 {
		return 0, ErrCanvasTrustInvalid
	}
	return entry.UserID, nil
}

func randomCanvasTrustToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate canvas trust token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func trimCanvasTrustToken(token string) string {
	return strings.TrimSpace(token)
}

func resolveCanvasTrustUser(userID int) (*model.User, error) {
	if userID <= 0 {
		return nil, ErrCanvasTrustUser
	}
	user, err := model.GetUserById(userID, false)
	if err != nil || user == nil || user.Id <= 0 {
		return nil, ErrCanvasTrustUser
	}
	if user.Status != common.UserStatusEnabled {
		return nil, errors.New("user is disabled")
	}
	return user, nil
}

func GetCanvasTrustUserProfile(userID int) (*CanvasTrustUserProfile, error) {
	user, err := resolveCanvasTrustUser(userID)
	if err != nil {
		return nil, err
	}
	serverAddr := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
	return &CanvasTrustUserProfile{
		ID:         user.Id,
		Username:   user.Username,
		Group:      user.Group,
		Quota:      user.Quota,
		UsedQuota:  user.UsedQuota,
		ServerAddr: serverAddr,
	}, nil
}

func ListCanvasTrustUserTokens(userID int, startIdx int, pageSize int) ([]CanvasTrustTokenItem, int, error) {
	if _, err := resolveCanvasTrustUser(userID); err != nil {
		return nil, 0, err
	}
	if pageSize <= 0 {
		pageSize = 100
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	tokens, err := model.GetAllUserTokens(userID, startIdx, pageSize)
	if err != nil {
		return nil, 0, err
	}
	total64, err := model.CountUserTokens(userID)
	if err != nil {
		return nil, 0, err
	}
	items := make([]CanvasTrustTokenItem, 0, len(tokens))
	for _, token := range tokens {
		if token == nil {
			continue
		}
		items = append(items, CanvasTrustTokenItem{
			ID:                 token.Id,
			Name:               token.Name,
			Key:                token.GetMaskedKey(),
			Status:             token.Status,
			RemainQuota:        token.RemainQuota,
			UnlimitedQuota:     token.UnlimitedQuota,
			ModelLimitsEnabled: token.ModelLimitsEnabled,
			ModelLimits:        token.ModelLimits,
			Group:              token.Group,
		})
	}
	return items, int(total64), nil
}

func GetCanvasTrustTokenKey(userID int, tokenID int) (string, error) {
	if _, err := resolveCanvasTrustUser(userID); err != nil {
		return "", err
	}
	token, err := model.GetTokenByIds(tokenID, userID)
	if err != nil {
		return "", err
	}
	if token.Status != common.TokenStatusEnabled {
		return "", errors.New("token is disabled")
	}
	key := strings.TrimSpace(token.GetFullKey())
	if key == "" {
		return "", errors.New("token key is empty")
	}
	if !strings.HasPrefix(key, "sk-") {
		key = "sk-" + key
	}
	return key, nil
}
