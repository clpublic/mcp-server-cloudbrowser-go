package config

type Config struct {
	Type       string `mapstructure:"type" json:"type" yaml:"type"`                      //模式
	Name       string `mapstructure:"name" json:"name" yaml:"name"`                      //appname
	Domain     string `mapstructure:"domain" json:"domain" yaml:"domain"`                //
	Host       string `mapstructure:"host" json:"host" yaml:"host"`                      //启动host
	Port       int    `mapstructure:"port" json:"port" yaml:"port"`                      //端口
	ServerHttp string `mapstructure:"server-http" json:"server-http" yaml:"server-http"` // 中心服务地址
	SignKey    string `mapstructure:"sign-key" json:"sign-key" yaml:"sign-key"`          // jwt签名
}

var Cfg = Config{}
