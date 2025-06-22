package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator"
	"github.com/spf13/viper"
)

type Conf struct {
	Port            int              `mapstructure:"port"  validate:"required"`
	UpstreamAddrs   []string         `mapstructure:"upstream_addrs"  validate:"required"`
	DomainResolvers []DomainResolver `mapstructure:"domain_resolvers"`
	Kafka           KafkaConf        `mapstructure:"kafka"`
}

type KafkaConf struct {
	Servers  string `mapstructure:"servers"`
	ClientId string `mapstructure:"client_id"`
	LogTopic string `mapstructure:"log_topic"`
}
type DomainResolver struct {
	Domain    string   `mapstructure:"domain"  validate:"required"`
	Resolvers []string `mapstructure:"resolvers"  validate:"required"`
}

func InitConf() (*Conf, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	conf := &Conf{}
	if err := v.Unmarshal(conf); err != nil {
		return nil, err
	}
	validate := validator.New()
	if err := validate.Struct(conf); err != nil {
		var sb strings.Builder
		for _, err := range err.(validator.ValidationErrors) {
			sb.WriteString(fmt.Sprintf("Field '%s' failed on '%s'\n", err.Field(), err.Tag()))
		}
		return nil, errors.New(sb.String())
	}

	return conf, nil
}
