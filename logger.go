package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	Message  string   `json:"message"`
}
type ILogger interface {
	Info(log Log) error
	Warn(log Log) error
	Error(log Log) error
	Debug(log Log) error
}
type LoggerType string

const (
	Kafka LoggerType = "kafka"
	File  LoggerType = "file"
)

type KafkaLogger struct {
	conf         *Conf
	mu           sync.Mutex
	producer     *kafka.Producer
	deliveryChan chan kafka.Event
}

func NewLogger(conf *Conf) ILogger {
	loggerType := conf.Log.Logger
	switch loggerType {
	case Kafka:
		logger := &KafkaLogger{
			conf:         conf,
			mu:           sync.Mutex{},
			deliveryChan: make(chan kafka.Event, 1000),
		}
		err := logger.Init()
		if err != nil {
			log.Fatalf("Failed to init logger %v", err)
			os.Exit(1)
		}
		return logger
	case File:
		file, err := os.OpenFile(conf.Log.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to init logger %v", err)
			os.Exit(1)
		}
		return &FileLogger{
			conf: conf,
			mu:   sync.Mutex{},
			file: file,
		}

	default:
		log.Fatalf("Invalid Logger Type")
		os.Exit(1)
	}
	return nil
}
func (l *KafkaLogger) Init() error {
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

func (l *KafkaLogger) write(level LogLevel, entry Log) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	topic := l.conf.Kafka.LogTopic

	err = l.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          data,
	}, l.deliveryChan)

	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	e := <-l.deliveryChan
	m := e.(*kafka.Message)
	if m.TopicPartition.Error != nil {
		return fmt.Errorf("failed to deliver message: %v", m.TopicPartition.Error)
	}

	return nil
}
func (l *KafkaLogger) Info(log Log) error {
	return l.write(Info, log)
}
func (l *KafkaLogger) Warn(log Log) error {
	return l.write(Warn, log)
}
func (l *KafkaLogger) Error(log Log) error {
	return l.write(Error, log)
}
func (l *KafkaLogger) Debug(log Log) error {
	return l.write(Debug, log)
}

type FileLogger struct {
	conf *Conf
	mu   sync.Mutex
	file *os.File
}

func (l *FileLogger) write(level LogLevel, entry Log) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	logEntry := fmt.Sprintf("[%s] At: %s - ClientIp: %s - Domain: %s , Resolver: %s - Question: %s\n", level, entry.Time, entry.ClientIp, entry.Domain, entry.Resolver, entry.Qtype)
	_, err := l.file.WriteString(logEntry)
	if err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}
	return nil
}
func (l *FileLogger) Info(log Log) error {
	return l.write(Info, log)
}
func (l *FileLogger) Warn(log Log) error {
	return l.write(Warn, log)
}
func (l *FileLogger) Error(log Log) error {
	return l.write(Error, log)
}
func (l *FileLogger) Debug(log Log) error {
	return l.write(Debug, log)
}
