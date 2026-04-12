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

	nowHour := time.Now().Truncate(time.Hour).Unix()
	nowDay := time.Now().Truncate(time.Hour * 24).Unix()
	nowWeek := time.Now().Truncate(time.Hour * 24 * 7).Unix()

	needUpdate := false

	// === 小时计数 ===
	if channel.ChannelInfo.HourlyCallResetTime < nowHour {
		channel.ChannelInfo.HourlyCallCount = 0
		channel.ChannelInfo.HourlyCallResetTime = nowHour
	}
	channel.ChannelInfo.HourlyCallCount++
	needUpdate = true

	// 检查小时限制
	if setting.HourlyCallLimit > 0 && channel.ChannelInfo.HourlyCallCount > setting.HourlyCallLimit {
		DisableChannel(types.ChannelError{
			ChannelId:   channelId,
			ChannelType: channel.Type,
			ChannelName: channel.Name,
			AutoBan:     true,
		}, fmt.Sprintf("%s小时调用次数已达上限（%d/%d）", ChannelDisableReasonPrefix, channel.ChannelInfo.HourlyCallCount, setting.HourlyCallLimit))
		return
	}

	// === 天计数 ===
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

	// === 周计数 ===
	if channel.ChannelInfo.WeeklyCallResetTime < nowWeek {
		channel.ChannelInfo.WeeklyCallCount = 0
		channel.ChannelInfo.WeeklyCallResetTime = nowWeek
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

// CheckChannelCallLimits 定时任务：重置过期的调用次数计数 + 自动启用过期限制的渠道
// 每分钟运行一次
func CheckChannelCallLimits() {
	for {
		time.Sleep(time.Minute)
		nowHour := time.Now().Truncate(time.Hour).Unix()
		nowDay := time.Now().Truncate(time.Hour * 24).Unix()
		nowWeek := time.Now().Truncate(time.Hour * 24 * 7).Unix()

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

			// 重置过期的计数
			if channel.ChannelInfo.HourlyCallResetTime < nowHour {
				channel.ChannelInfo.HourlyCallCount = 0
				channel.ChannelInfo.HourlyCallResetTime = nowHour
				needSave = true
			}
			if channel.ChannelInfo.DailyCallResetTime < nowDay {
				channel.ChannelInfo.DailyCallCount = 0
				channel.ChannelInfo.DailyCallResetTime = nowDay
				needSave = true
			}
			if channel.ChannelInfo.WeeklyCallResetTime < nowWeek {
				channel.ChannelInfo.WeeklyCallCount = 0
				channel.ChannelInfo.WeeklyCallResetTime = nowWeek
				needSave = true
			}

			// 自动启用：如果渠道是因为调用限制被禁用的，且限制周期已过，则自动启用
			if channel.Status == common.ChannelStatusAutoDisabled {
				// 获取禁用原因
				otherInfo := channel.GetOtherInfo()
				statusReason, _ := otherInfo["status_reason"].(string)

				if isChannelLimitDisabled(statusReason) {
					// 检查是否所有过期的限制都已重置
					canEnable := true
					if setting.HourlyCallLimit > 0 && channel.ChannelInfo.HourlyCallResetTime < nowHour {
						canEnable = true // 小时限制已重置
					}
					if setting.DailyCallLimit > 0 && channel.ChannelInfo.DailyCallResetTime < nowDay {
						canEnable = true // 天限制已重置
					}
					if setting.WeeklyCallLimit > 0 && channel.ChannelInfo.WeeklyCallResetTime < nowWeek {
						canEnable = true // 周限制已重置
					}

					if canEnable {
						EnableChannel(channel.Id, "", channel.Name)
						// 清空计数和禁用原因
						channel.ChannelInfo.HourlyCallCount = 0
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
