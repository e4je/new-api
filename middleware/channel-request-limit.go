package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const (
	channelRequestLimitHourWindowSeconds = int64(60 * 60)
	channelRequestLimitRedisPrefix       = "rateLimit:channel"
)

type channelRequestLimitMemoryState struct {
	HourTimestamps  []int64
	DayBucketStart  int64
	DayCount        int
	WeekBucketStart int64
	WeekCount       int
}

var channelRequestLimitMemoryStore = map[int]*channelRequestLimitMemoryState{}
var channelRequestLimitMemoryLock sync.Mutex

func normalizeChannelLimit(limit int) int {
	if limit < 0 {
		return 0
	}
	return limit
}

func currentDayBucketStart(now time.Time) int64 {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return dayStart.Unix()
}

func currentWeekBucketStart(now time.Time) int64 {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekday := int(dayStart.Weekday()) // Sunday=0
	if weekday == 0 {
		weekday = 7
	}
	weekStart := dayStart.AddDate(0, 0, -(weekday - 1))
	return weekStart.Unix()
}

func nextDayBucketStart(now time.Time) int64 {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return dayStart.AddDate(0, 0, 1).Unix()
}

func nextWeekBucketStart(now time.Time) int64 {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekday := int(dayStart.Weekday()) // Sunday=0
	if weekday == 0 {
		weekday = 7
	}
	daysUntil := 8 - weekday
	return dayStart.AddDate(0, 0, daysUntil).Unix()
}

func allowChannelRequestLimitWithMemory(channelId int, hourlyLimit, dailyLimit, weeklyLimit int, now time.Time) (bool, string) {
	channelRequestLimitMemoryLock.Lock()
	defer channelRequestLimitMemoryLock.Unlock()

	state, ok := channelRequestLimitMemoryStore[channelId]
	if !ok {
		state = &channelRequestLimitMemoryState{
			HourTimestamps: make([]int64, 0, 16),
		}
		channelRequestLimitMemoryStore[channelId] = state
	}

	nowUnix := now.Unix()

	if hourlyLimit > 0 {
		cutoff := nowUnix - channelRequestLimitHourWindowSeconds
		kept := state.HourTimestamps[:0]
		for _, ts := range state.HourTimestamps {
			if ts > cutoff {
				kept = append(kept, ts)
			}
		}
		state.HourTimestamps = kept
		if len(state.HourTimestamps) >= hourlyLimit {
			return false, fmt.Sprintf("channel request limit reached: hourly window (%d)", hourlyLimit)
		}
		state.HourTimestamps = append(state.HourTimestamps, nowUnix)
	}

	if dailyLimit > 0 {
		dayBucket := currentDayBucketStart(now)
		if state.DayBucketStart != dayBucket {
			state.DayBucketStart = dayBucket
			state.DayCount = 0
		}
		if state.DayCount >= dailyLimit {
			return false, fmt.Sprintf("channel request limit reached: daily window (%d)", dailyLimit)
		}
		state.DayCount++
	}

	if weeklyLimit > 0 {
		weekBucket := currentWeekBucketStart(now)
		if state.WeekBucketStart != weekBucket {
			state.WeekBucketStart = weekBucket
			state.WeekCount = 0
		}
		if state.WeekCount >= weeklyLimit {
			return false, fmt.Sprintf("channel request limit reached: weekly window (%d)", weeklyLimit)
		}
		state.WeekCount++
	}

	return true, ""
}

func allowChannelRequestLimitWithRedis(ctx context.Context, channelId int, hourlyLimit, dailyLimit, weeklyLimit int, now time.Time) (bool, string, error) {
	rdb := common.RDB
	nowUnix := now.Unix()
	nowNano := now.UnixNano()

	if hourlyLimit > 0 {
		hourKey := fmt.Sprintf("%s:%d:hour", channelRequestLimitRedisPrefix, channelId)
		cutoff := nowUnix - channelRequestLimitHourWindowSeconds

		if _, err := rdb.ZRemRangeByScore(ctx, hourKey, "-inf", fmt.Sprintf("%d", cutoff)).Result(); err != nil {
			return false, "", err
		}
		count, err := rdb.ZCard(ctx, hourKey).Result()
		if err != nil {
			return false, "", err
		}
		if count >= int64(hourlyLimit) {
			return false, fmt.Sprintf("channel request limit reached: hourly window (%d)", hourlyLimit), nil
		}
		member := fmt.Sprintf("%d:%d", nowNano, count+1)
		if _, err = rdb.ZAdd(ctx, hourKey, &redis.Z{
			Score:  float64(nowUnix),
			Member: member,
		}).Result(); err != nil {
			return false, "", err
		}
		if _, err = rdb.Expire(ctx, hourKey, (time.Duration(channelRequestLimitHourWindowSeconds)+120)*time.Second).Result(); err != nil {
			return false, "", err
		}
	}

	if dailyLimit > 0 {
		dayBucket := currentDayBucketStart(now)
		dayKey := fmt.Sprintf("%s:%d:day:%d", channelRequestLimitRedisPrefix, channelId, dayBucket)
		count, err := rdb.Incr(ctx, dayKey).Result()
		if err != nil {
			return false, "", err
		}
		if count == 1 {
			expireSeconds := nextDayBucketStart(now) - nowUnix + 120
			if expireSeconds <= 0 {
				expireSeconds = 120
			}
			if _, err = rdb.Expire(ctx, dayKey, time.Duration(expireSeconds)*time.Second).Result(); err != nil {
				return false, "", err
			}
		}
		if count > int64(dailyLimit) {
			return false, fmt.Sprintf("channel request limit reached: daily window (%d)", dailyLimit), nil
		}
	}

	if weeklyLimit > 0 {
		weekBucket := currentWeekBucketStart(now)
		weekKey := fmt.Sprintf("%s:%d:week:%d", channelRequestLimitRedisPrefix, channelId, weekBucket)
		count, err := rdb.Incr(ctx, weekKey).Result()
		if err != nil {
			return false, "", err
		}
		if count == 1 {
			expireSeconds := nextWeekBucketStart(now) - nowUnix + 120
			if expireSeconds <= 0 {
				expireSeconds = 120
			}
			if _, err = rdb.Expire(ctx, weekKey, time.Duration(expireSeconds)*time.Second).Result(); err != nil {
				return false, "", err
			}
		}
		if count > int64(weeklyLimit) {
			return false, fmt.Sprintf("channel request limit reached: weekly window (%d)", weeklyLimit), nil
		}
	}

	return true, "", nil
}

func ChannelRequestRateLimit() func(c *gin.Context) {
	return func(c *gin.Context) {
		channelId := common.GetContextKeyInt(c, constant.ContextKeyChannelId)
		if channelId <= 0 {
			c.Next()
			return
		}

		otherSetting, ok := common.GetContextKeyType[dto.ChannelOtherSettings](c, constant.ContextKeyChannelOtherSetting)
		if !ok {
			c.Next()
			return
		}

		hourlyLimit := normalizeChannelLimit(otherSetting.ChannelRateLimitHourly)
		dailyLimit := normalizeChannelLimit(otherSetting.ChannelRateLimitDaily)
		weeklyLimit := normalizeChannelLimit(otherSetting.ChannelRateLimitWeekly)
		if hourlyLimit == 0 && dailyLimit == 0 && weeklyLimit == 0 {
			c.Next()
			return
		}

		now := time.Now()
		if common.RedisEnabled && common.RDB != nil {
			allowed, message, err := allowChannelRequestLimitWithRedis(context.Background(), channelId, hourlyLimit, dailyLimit, weeklyLimit, now)
			if err != nil {
				abortWithOpenAiMessage(c, http.StatusInternalServerError, "channel_rate_limit_check_failed")
				return
			}
			if !allowed {
				abortWithOpenAiMessage(c, http.StatusTooManyRequests, message)
				return
			}
			c.Next()
			return
		}

		allowed, message := allowChannelRequestLimitWithMemory(channelId, hourlyLimit, dailyLimit, weeklyLimit, now)
		if !allowed {
			abortWithOpenAiMessage(c, http.StatusTooManyRequests, message)
			return
		}
		c.Next()
	}
}
