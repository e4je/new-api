package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetChannelUsageForUser 普通用户查看渠道用量信息
func GetChannelUsageForUser(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiError(c, nil)
		return
	}

	// 获取所有启用的渠道
	channels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	type ChannelUsageInfo struct {
		Id              int    `json:"id"`
		Name            string `json:"name"`
		Status          int    `json:"status"`
		HourlyLimit     int    `json:"hourly_limit"`
		HourlyRemaining int    `json:"hourly_remaining"`
		DailyLimit      int    `json:"daily_limit"`
		DailyRemaining  int    `json:"daily_remaining"`
		WeeklyLimit     int    `json:"weekly_limit"`
		WeeklyRemaining int    `json:"weekly_remaining"`
	}

	usageInfos := make([]ChannelUsageInfo, 0)
	for _, channel := range channels {
		if channel.Status != common.ChannelStatusEnabled {
			continue
		}

		setting := channel.GetSetting()
		hourlyLimit := setting.HourlyCallLimit
		dailyLimit := setting.DailyCallLimit
		weeklyLimit := setting.WeeklyCallLimit

		// 只返回有限制的渠道
		if hourlyLimit == 0 && dailyLimit == 0 && weeklyLimit == 0 {
			continue
		}

		hourlyCount := len(channel.ChannelInfo.HourlyCallTimestamps)
		dailyCount := channel.ChannelInfo.DailyCallCount
		weeklyCount := channel.ChannelInfo.WeeklyCallCount

		hourlyRemaining := 0
		if hourlyLimit > 0 {
			hourlyRemaining = max(0, hourlyLimit-hourlyCount)
		}
		dailyRemaining := 0
		if dailyLimit > 0 {
			dailyRemaining = max(0, dailyLimit-dailyCount)
		}
		weeklyRemaining := 0
		if weeklyLimit > 0 {
			weeklyRemaining = max(0, weeklyLimit-weeklyCount)
		}

		usageInfos = append(usageInfos, ChannelUsageInfo{
			Id:              channel.Id,
			Name:            channel.Name,
			Status:          channel.Status,
			HourlyLimit:     hourlyLimit,
			HourlyRemaining: hourlyRemaining,
			DailyLimit:      dailyLimit,
			DailyRemaining:  dailyRemaining,
			WeeklyLimit:     weeklyLimit,
			WeeklyRemaining: weeklyRemaining,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usageInfos,
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
