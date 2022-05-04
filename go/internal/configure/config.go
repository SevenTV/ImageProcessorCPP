package configure

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func checkErr(err error) {
	if err != nil {
		zap.S().Fatalw("config",
			"error", err,
		)
	}
}

func New() *Config {
	initLogging("info")

	config := viper.New()

	// Default config
	b, _ := json.Marshal(Config{
		ConfigFile: "config.yaml",
	})
	tmp := viper.New()
	defaultConfig := bytes.NewReader(b)
	tmp.SetConfigType("json")
	checkErr(tmp.ReadConfig(defaultConfig))
	checkErr(config.MergeConfigMap(viper.AllSettings()))

	pflag.String("mode", "", "The running mode, `controller` or `edge`")
	pflag.String("config", "config.yaml", "Config file location")
	pflag.Bool("noheader", false, "Disable the startup header")

	pflag.Parse()
	checkErr(config.BindPFlags(pflag.CommandLine))

	// File
	config.SetConfigFile(config.GetString("config"))
	config.AddConfigPath(".")
	if err := config.ReadInConfig(); err == nil {
		checkErr(config.MergeInConfig())
	}

	bindEnvs(config, Config{})

	// Environment
	config.AutomaticEnv()
	config.SetEnvPrefix("IP")
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AllowEmptyEnv(true)

	// Print final config
	c := &Config{}
	checkErr(config.Unmarshal(&c))

	initLogging(c.Level)

	return c
}

func bindEnvs(config *viper.Viper, iface interface{}, parts ...string) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)
	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		t := ift.Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}
		switch v.Kind() {
		case reflect.Struct:
			bindEnvs(config, v.Interface(), append(parts, tv)...)
		default:
			_ = config.BindEnv(strings.Join(append(parts, tv), "."))
		}
	}
}

type Config struct {
	Level      string `mapstructure:"level" json:"level"`
	ConfigFile string `mapstructure:"config" json:"config"`
	NoHeader   bool   `mapstructure:"noheader" json:"noheader"`

	Worker struct {
		Jobs    int    `mapstructure:"jobs" json:"jobs"`
		TempDir string `mapstructure:"temp_dir" json:"temp_dir"`
	} `mapstructure:"worker" json:"worker"`

	Health struct {
		Bind    string `mapstructure:"bind" json:"bind"`
		Enabled bool   `mapstructure:"enabled" json:"enabled"`
	} `mapstructure:"health" json:"health"`

	KubeMQ struct {
		Host      string `mapstructure:"host" json:"host"`
		Port      int    `mapstructure:"port" json:"port"`
		ClientId  string `mapstructure:"client_id" json:"client_id"`
		AuthToken string `mapstructure:"auth_token" json:"auth_token"`
	} `mapstructure:"kubemq" json:"kubemq"`

	S3 struct {
		Region      string `mapstructure:"region" json:"region"`
		Endpoint    string `mapstructure:"endpoint" json:"endpoint"`
		AccessToken string `mapstructure:"access_token" json:"access_token"`
		SecretKey   string `mapstructure:"secret_key" json:"secret_key"`
	} `mapstructure:"s3" json:"s3"`

	Monitoring struct {
		Bind    string `mapstructure:"bind" json:"bind"`
		Enabled bool   `mapstructure:"enabled" json:"enabled"`
		Labels  Labels `mapstructure:"labels" json:"labels"`
	} `mapstructure:"monitoring" json:"monitoring"`
}

type Labels []struct {
	Key   string `mapstructure:"key" json:"key"`
	Value string `mapstructure:"value" json:"value"`
}

func (l Labels) ToPrometheus() prometheus.Labels {
	mp := prometheus.Labels{}

	for _, v := range l {
		mp[v.Key] = v.Value
	}

	return mp
}
