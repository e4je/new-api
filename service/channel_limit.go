package service

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
)

// IncrementChannelCallCount 增加渠道调用次数计数，并检查是否超出限制
// 每次渠道成功处理请求后调用
func IncrementChannelCallCount(channelId int) {
	channel, err := model.CacheGetChannel(channelId)
	if err != nil {
		return
	}

	setting := channel.GetSetting()
	if setting.HourlyCallLimit == 0 && setting.WeeklyCallLimit == 0 {
		return // 没有限制，直接返回
	}

	nowHour := time.Now().Truncate(time.Hour).Unix()
	nowWeek := time.Now().Truncate(time.Hour * 24 * 7).Unix() // 简化：按7天窗口

	needUpdate := false

	// === 小时计数 ===
	if channel.ChannelInfo.HourlyCallResetTime < nowHour {
		// 新的小时窗口，重置计数
		channel.ChannelInfo.HourlyCallCount = 0
		channel.ChannelInfo.HourlyCallResetTime = nowHour
		needUpdate = true
	}
	channel.ChannelInfo.HourlyCallCount++

	// 检查小时限制
	if setting.HourlyCallLimit > 0 && channel.ChannelInfo.HourlyCallCount > setting.HourlyCallLimit {
		DisableChannel(types.ChannelError{
			ChannelId:   channelId,
			ChannelType: channel.Type,
			ChannelName: channel.Name,
			AutoBan:     true,
		}, fmt.Sprintf("小时调用次数已达上限（%d/%d）", channel.ChannelInfo.HourlyCallCount, setting.HourlyCallLimit))
		return
	}

	// === 周计数 ===
	if channel.ChannelInfo.WeeklyCallResetTime < nowWeek {
		// 新的周窗口，重置计数
		channel.ChannelInfo.WeeklyCallCount = 0
		channel.ChannelInfo.WeeklyCallResetTime = nowWeek
		needUpdate = true
	}
	channel.ChannelInfo.WeeklyCallCount++

	// 检查周限制
	if setting.WeeklyCallLimit > 0 && channel.ChannelInfo.WeeklyCallCount > setting.WeeklyCallLimit {
		DisableChannel(types.ChannelError{
			ChannelId:   channelId,
			ChannelType: channel.Type,
			ChannelName: channel.Name,
			AutoBan:     true,
		}, fmt.Sprintf("周调用次数已达上限（%d/%d）", channel.ChannelInfo.WeeklyCallCount, setting.WeeklyCallLimit))
		return
	}

	if needUpdate {
		_ = channel.SaveChannelInfo()
	}
}

// CheckChannelCallLimits 定时任务：重置过期的调用次数计数
// 每分钟运行一次
func CheckChannelCallLimits() {
	for {
		time.Sleep(time.Minute)
		nowHour := time.Now().Truncate(time.Hour).Unix()
		nowWeek := time.Now().Truncate(time.Hour * 24 * 7).Unix()

		channels, err := model.GetAllChannels(0, 0, true, true)
		if err != nil {
			continue
		}

		for _, channel := range channels {
			if channel.Status != common.ChannelStatusEnabled {
				continue
			}
			setting := channel.GetSetting()
			if setting.HourlyCallLimit == 0 && setting.WeeklyCallLimit == 0 {
				continue
			}

			needSave := false
			if channel.ChannelInfo.HourlyCallResetTime < nowHour {
				channel.ChannelInfo.HourlyCallCount = 0
				channel.ChannelInfo.HourlyCallResetTime = nowHour
				needSave = true
			}
			if channel.ChannelInfo.WeeklyCallResetTime < nowWeek {
				channel.ChannelInfo.WeeklyCallCount = 0
				channel.ChannelInfo.WeeklyCallResetTime = nowWeek
				needSave = true
			}

			if needSave {
				_ = channel.SaveChannelInfo()
			}
		}

		common.SysLog("channel call limits check done")
	}
}
