package system_setting

import "github.com/QuantumNous/new-api/setting/config"

type AliyunCaptchaSettings struct {
	Enabled   bool   `json:"enabled"`
	Region    string `json:"region"`
	Prefix    string `json:"prefix"`
	SceneId   string `json:"scene_id"`
	Mode      string `json:"mode"`
	ScriptUrl string `json:"script_url"`
}

var defaultAliyunCaptchaSettings = AliyunCaptchaSettings{
	Enabled:   false,
	Region:    "cn",
	Prefix:    "",
	SceneId:   "",
	Mode:      "popup",
	ScriptUrl: "https://o.alicdn.com/captcha-frontend/aliyunCaptcha/AliyunCaptcha.js",
}

func init() {
	config.GlobalConfig.Register("aliyun_captcha", &defaultAliyunCaptchaSettings)
}

func GetAliyunCaptchaSettings() *AliyunCaptchaSettings {
	return &defaultAliyunCaptchaSettings
}
