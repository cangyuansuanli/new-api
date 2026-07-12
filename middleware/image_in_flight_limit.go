package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

type imageInFlightCounter struct {
	mu     sync.Mutex
	global int
	users  map[int]int
}

var activeImageRequests = imageInFlightCounter{users: make(map[int]int)}

const imageInFlightAcquireScript = `
local global_key = KEYS[1]
local user_key = KEYS[2]
local now = tonumber(ARGV[1])
local expires_at = tonumber(ARGV[2])
local global_limit = tonumber(ARGV[3])
local user_limit = tonumber(ARGV[4])
local token = ARGV[5]
local ttl = tonumber(ARGV[6])
redis.call('ZREMRANGEBYSCORE', global_key, '-inf', now)
redis.call('ZREMRANGEBYSCORE', user_key, '-inf', now)
if global_limit > 0 and redis.call('ZCARD', global_key) >= global_limit then return 0 end
if user_limit > 0 and redis.call('ZCARD', user_key) >= user_limit then return 0 end
redis.call('ZADD', global_key, expires_at, token)
redis.call('ZADD', user_key, expires_at, token)
redis.call('EXPIRE', global_key, ttl)
redis.call('EXPIRE', user_key, ttl)
return 1
`

const imageInFlightReleaseScript = `
redis.call('ZREM', KEYS[1], ARGV[1])
redis.call('ZREM', KEYS[2], ARGV[1])
return 1
`

type imageInFlightLease struct {
	globalKey string
	userKey   string
	token     string
}

func tryAcquireDistributedImageLease(c *gin.Context, userID, perUserLimit, globalLimit int) (*imageInFlightLease, bool, error) {
	requestID := c.GetString(common.RequestIdKey)
	if requestID == "" {
		random, _ := common.GenerateRandomCharsKey(24)
		requestID = random
		if requestID == "" {
			requestID = fmt.Sprintf("fallback-%d", time.Now().UnixNano())
		}
	}
	token := fmt.Sprintf("%d:%s", userID, requestID)
	globalKey := "new-api:image-flight:{image}:global"
	userKey := fmt.Sprintf("new-api:image-flight:{image}:user:%d", userID)
	leaseSeconds := common.GetEnvOrDefault("IMAGE_MAX_IN_FLIGHT_LEASE_SECONDS", 3600)
	now := time.Now().UnixMilli()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	result, err := common.RDB.Eval(
		ctx,
		imageInFlightAcquireScript,
		[]string{globalKey, userKey},
		now,
		now+int64(leaseSeconds)*1000,
		globalLimit,
		perUserLimit,
		token,
		leaseSeconds+60,
	).Int()
	if err != nil {
		return nil, false, err
	}
	return &imageInFlightLease{globalKey: globalKey, userKey: userKey, token: token}, result == 1, nil
}

func (lease *imageInFlightLease) release() {
	if lease == nil || common.RDB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = common.RDB.Eval(ctx, imageInFlightReleaseScript, []string{lease.globalKey, lease.userKey}, lease.token).Err()
}

func (l *imageInFlightCounter) tryAcquire(userID, perUserLimit, globalLimit int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if globalLimit > 0 && l.global >= globalLimit {
		return false
	}
	if perUserLimit > 0 && l.users[userID] >= perUserLimit {
		return false
	}

	l.global++
	l.users[userID]++
	return true
}

func (l *imageInFlightCounter) release(userID int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.global > 0 {
		l.global--
	}
	if count := l.users[userID]; count <= 1 {
		delete(l.users, userID)
	} else {
		l.users[userID] = count - 1
	}
}

func isImageRelayRequest(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}
	switch c.Request.URL.Path {
	case "/v1/images/generations", "/v1/images/edits", "/v1/edits":
		return true
	default:
		return false
	}
}

// ImageInFlightLimit bounds active image generation/edit requests before any
// middleware parses or copies a potentially large multipart request body.
func ImageInFlightLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isImageRelayRequest(c) {
			c.Next()
			return
		}

		perUserLimit := constant.ImageMaxInFlightPerUser
		globalLimit := constant.ImageMaxInFlightGlobal
		if perUserLimit <= 0 && globalLimit <= 0 {
			c.Next()
			return
		}

		userID := c.GetInt("id")
		if common.RedisEnabled && common.RDB != nil {
			lease, admitted, err := tryAcquireDistributedImageLease(c, userID, perUserLimit, globalLimit)
			if err == nil {
				if !admitted {
					abortWithOpenAiMessage(c, http.StatusTooManyRequests, "too many concurrent image requests; please retry later")
					return
				}
				defer lease.release()
				c.Next()
				return
			}
			common.SysError("distributed image in-flight limiter unavailable: " + err.Error())
		}
		if !activeImageRequests.tryAcquire(userID, perUserLimit, globalLimit) {
			abortWithOpenAiMessage(c, http.StatusTooManyRequests, "too many concurrent image requests; please retry later")
			return
		}
		defer activeImageRequests.release(userID)

		c.Next()
	}
}
