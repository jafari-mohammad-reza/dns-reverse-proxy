package main

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type LogLevel string

const (
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
	Debug LogLevel = "debug"
)

type Log struct {
	Time     string   `json:"time"`
	Level    LogLevel `json:"level"`
	Domain   string   `json:"domain"`
	ClientIp string   `json:"client_ip"`
	Qtype    string   `json:"qtype"`
	Resolver string   `json:"resolver"`
}
type Logger struct {
	conf     *Conf
	mu       sync.Mutex
	producer *kafka.Producer
}

func NewLogger(conf *Conf) *Logger {
	return &Logger{
		conf: conf,
		mu:   sync.Mutex{},
	}
}
func (l *Logger) Init() error {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": l.conf.Kafka.Servers,
		"client.id":         l.conf.Kafka.ClientId,
		"acks":              "all"})

	if err != nil {
		return fmt.Errorf("failed to create producer: %s", err)
	}
	l.producer = p
	return nil
}

func (l *Logger) write(level LogLevel, entry Log) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	topic := l.conf.Kafka.LogTopic
	deliveryChan := make(chan kafka.Event, 1)

	err = l.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          data,
	}, deliveryChan)

	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	e := <-deliveryChan
	m := e.(*kafka.Message)
	if m.TopicPartition.Error != nil {
		return fmt.Errorf("failed to deliver message: %v", m.TopicPartition.Error)
	}

	return nil
}
func (l *Logger) Info(log Log) error {
	return l.write(Info, log)
}
func (l *Logger) Warn(log Log) error {
	return l.write(Warn, log)
}
func (l *Logger) Error(log Log) error {
	return l.write(Error, log)
}
func (l *Logger) Debug(log Log) error {
	return l.write(Debug, log)
}
