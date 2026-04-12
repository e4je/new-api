package service

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
)

const ChannelDisableReasonPrefix = "channel_limit:"

// IncrementChannelCallCount 增加渠道调用次数计数，并检查是否超出限制
// 每次渠道成功处理请求后调用
func IncrementChannelCallCount(channelId int) {
	channel, err := model.CacheGetChannel(channelId)
	if err != nil {
		return
	}

	setting := channel.GetSetting()
	if setting.HourlyCallLimit == 0 && setting.DailyCallLimit == 0 && setting.WeeklyCallLimit == 0 {
		return // 没有限制，直接返回
	}

	now := time.Now().Unix()
	nowDay := time.Now().Truncate(time.Hour * 24).Unix()
	// 周一 00:00：time.Weekday(1) = Monday
	weekday := time.Now().Weekday()
	daysSinceMonday := (int(weekday) - 1 + 7) % 7
	mondayMidnight := time.Now().Truncate(time.Hour * 24).AddDate(0, 0, -daysSinceMonday).Unix()

	needUpdate := false

	// === 小时计数（滑动窗口：记录每次调用时间戳） ===
	if channel.ChannelInfo.HourlyCallTimestamps == nil {
		channel.ChannelInfo.HourlyCallTimestamps = []int64{}
	}
	// 清理超过1小时的旧记录
	oneHourAgo := now - 3600
	validTimestamps := make([]int64, 0, len(channel.ChannelInfo.HourlyCallTimestamps)+1)
	for _, ts := range channel.ChannelInfo.HourlyCallTimestamps {
		if ts > oneHourAgo {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	validTimestamps = append(validTimestamps, now)
	channel.ChannelInfo.HourlyCallTimestamps = validTimestamps
	hourlyCount := len(validTimestamps)
	needUpdate = true

	// 检查小时限制
	if setting.HourlyCallLimit > 0 && hourlyCount > setting.HourlyCallLimit {
		DisableChannel(types.ChannelError{
			ChannelId:   channelId,
			ChannelType: channel.Type,
			ChannelName: channel.Name,
			AutoBan:     true,
		}, fmt.Sprintf("%s小时调用次数已达上限（%d/%d）", ChannelDisableReasonPrefix, hourlyCount, setting.HourlyCallLimit))
		return
	}

	// === 天计数（固定窗口：每天00:00重置） ===
	if channel.ChannelInfo.DailyCallResetTime < nowDay {
		channel.ChannelInfo.DailyCallCount = 0
		channel.ChannelInfo.DailyCallResetTime = nowDay
	}
	channel.ChannelInfo.DailyCallCount++
	needUpdate = true

	// 检查天限制
	if setting.DailyCallLimit > 0 && channel.ChannelInfo.DailyCallCount > setting.DailyCallLimit {
		DisableChannel(types.ChannelError{
			ChannelId:   channelId,
			ChannelType: channel.Type,
			ChannelName: channel.Name,
			AutoBan:     true,
		}, fmt.Sprintf("%s天调用次数已达上限（%d/%d）", ChannelDisableReasonPrefix, channel.ChannelInfo.DailyCallCount, setting.DailyCallLimit))
		return
	}

	// === 周计数（固定窗口：每周一00:00重置） ===
	if channel.ChannelInfo.WeeklyCallResetTime < mondayMidnight {
		channel.ChannelInfo.WeeklyCallCount = 0
		channel.ChannelInfo.WeeklyCallResetTime = mondayMidnight
	}
	channel.ChannelInfo.WeeklyCallCount++
	needUpdate = true

	// 检查周限制
	if setting.WeeklyCallLimit > 0 && channel.ChannelInfo.WeeklyCallCount > setting.WeeklyCallLimit {
		DisableChannel(types.ChannelError{
			ChannelId:   channelId,
			ChannelType: channel.Type,
			ChannelName: channel.Name,
			AutoBan:     true,
		}, fmt.Sprintf("%s周调用次数已达上限（%d/%d）", ChannelDisableReasonPrefix, channel.ChannelInfo.WeeklyCallCount, setting.WeeklyCallLimit))
		return
	}

	if needUpdate {
		_ = channel.SaveChannelInfo()
	}
}

// isChannelLimitDisabled 检查渠道是否是因为调用限制被禁用的
func isChannelLimitDisabled(reason string) bool {
	return len(reason) > len(ChannelDisableReasonPrefix) && reason[:len(ChannelDisableReasonPrefix)] == ChannelDisableReasonPrefix
}

// CheckChannelCallLimits 定时任务：清理过期的调用次数计数 + 自动启用过期限制的渠道
// 每分钟运行一次
func CheckChannelCallLimits() {
	for {
		time.Sleep(time.Minute)
		now := time.Now().Unix()
		nowDay := time.Now().Truncate(time.Hour * 24).Unix()
		// 周一 00:00
		weekday := time.Now().Weekday()
		daysSinceMonday := (int(weekday) - 1 + 7) % 7
		mondayMidnight := time.Now().Truncate(time.Hour * 24).AddDate(0, 0, -daysSinceMonday).Unix()

		channels, err := model.GetAllChannels(0, 0, true, true)
		if err != nil {
			continue
		}

		for _, channel := range channels {
			setting := channel.GetSetting()
			if setting.HourlyCallLimit == 0 && setting.DailyCallLimit == 0 && setting.WeeklyCallLimit == 0 {
				continue
			}

			needSave := false
			oneHourAgo := now - 3600

			// 清理小时滑动窗口的过期记录
			if len(channel.ChannelInfo.HourlyCallTimestamps) > 0 {
				validTimestamps := make([]int64, 0, len(channel.ChannelInfo.HourlyCallTimestamps))
				for _, ts := range channel.ChannelInfo.HourlyCallTimestamps {
					if ts > oneHourAgo {
						validTimestamps = append(validTimestamps, ts)
					}
				}
				if len(validTimestamps) != len(channel.ChannelInfo.HourlyCallTimestamps) {
					channel.ChannelInfo.HourlyCallTimestamps = validTimestamps
					needSave = true
				}
			}

			// 重置过期的天计数
			if channel.ChannelInfo.DailyCallResetTime < nowDay {
				channel.ChannelInfo.DailyCallCount = 0
				channel.ChannelInfo.DailyCallResetTime = nowDay
				needSave = true
			}

			// 重置过期的周计数
			if channel.ChannelInfo.WeeklyCallResetTime < mondayMidnight {
				channel.ChannelInfo.WeeklyCallCount = 0
				channel.ChannelInfo.WeeklyCallResetTime = mondayMidnight
				needSave = true
			}

			// 自动启用：如果渠道是因为调用限制被禁用的，且限制周期已过，则自动启用
			if channel.Status == common.ChannelStatusAutoDisabled {
				otherInfo := channel.GetOtherInfo()
				statusReason, _ := otherInfo["status_reason"].(string)

				if isChannelLimitDisabled(statusReason) {
					// 计算当前有效计数
					hourlyCount := len(channel.ChannelInfo.HourlyCallTimestamps)
					dailyCount := channel.ChannelInfo.DailyCallCount
					weeklyCount := channel.ChannelInfo.WeeklyCallCount

					canEnable := true
					if setting.HourlyCallLimit > 0 && hourlyCount >= setting.HourlyCallLimit {
						canEnable = false
					}
					if setting.DailyCallLimit > 0 && dailyCount >= setting.DailyCallLimit {
						canEnable = false
					}
					if setting.WeeklyCallLimit > 0 && weeklyCount >= setting.WeeklyCallLimit {
						canEnable = false
					}

					if canEnable {
						EnableChannel(channel.Id, "", channel.Name)
						channel.ChannelInfo.HourlyCallTimestamps = []int64{}
						channel.ChannelInfo.DailyCallCount = 0
						channel.ChannelInfo.WeeklyCallCount = 0
						delete(otherInfo, "status_reason")
						delete(otherInfo, "status_time")
						channel.SetOtherInfo(otherInfo)
						channel.Status = common.ChannelStatusEnabled
						needSave = true
					}
				}
			}

			if needSave {
				_ = channel.SaveChannelInfo()
			}
		}

		common.SysLog("channel call limits check done")
	}
}
