package config

type RecoveringSlurmJobConfig struct {
	Image    string
	Schedule string
}

var RecoveringJobConfig = RecoveringSlurmJobConfig{
	Image:    "registry.cn-hangzhou.aliyuncs.com/troila-klcloud/ubuntu-recovering:24.04",
	Schedule: "*/5 * * * *",
}
