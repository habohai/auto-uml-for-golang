package logging

import (
	"fmt"
	"time"

	"github.com/haibeichina/gin_video_server/web/pkg/setting"
	"github.com/spf13/viper"
)

func getLogFilePath() string {
	return fmt.Sprintf("%s%s",
		viper.GetString("runtime.path"),
		viper.GetString("runtime.log.path"),
	)
}

func getLogFileName() string {
	return fmt.Sprintf("%s%s.%s",
		viper.GetString("runtime.log.name"),
		time.Now().Format(setting.AppSetting.TimeFormat),
		viper.GetString("runtime.log.ext"),
	)
}
