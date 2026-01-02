package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/tnqbao/gau-upload-service/consumer/service"
	"github.com/tnqbao/gau-upload-service/consumer/topic"
	"github.com/tnqbao/gau-upload-service/shared/config"
	"github.com/tnqbao/gau-upload-service/shared/infra"
)

const (
	QueueName   = "upload.chunked"
	ConsumerTag = "gau-upload-consumer"
)

func main() {
	// Load environment variables
	if err := godotenv.Load("/gau_upload/upload.env"); err != nil {
		log.Println("No .env file found, continuing with environment variables")
	}

	// Initialize configuration
	cfg := config.NewConfig()
	log.Printf("Consumer service starting with config: %+v", cfg.EnvConfig.Environment)

	// Initialize infrastructure for consumer (requires RabbitMQ)
	inf := infra.InitInfraForConsumer(cfg)
	defer func() {
		if inf.RabbitMQ != nil {
			inf.RabbitMQ.Close()
		}
	}()

	// Create chunker service
	chunkerService := service.NewChunkerService(cfg, inf)

	// Create topic handler
	handler := topic.NewChunkedUploadHandler(inf, chunkerService)

	// Declare queue
	if err := inf.RabbitMQ.DeclareQueue(QueueName, true, false); err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// Start consuming messages
	msgs, err := inf.RabbitMQ.Consume(QueueName, ConsumerTag)
	if err != nil {
		log.Fatalf("Failed to start consuming: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start message processing in goroutine
	go func() {
		log.Println("Consumer started. Waiting for messages...")
		for {
			select {
			case <-ctx.Done():
				log.Println("Context cancelled, stopping consumer...")
				return
			case msg, ok := <-msgs:
				if !ok {
					log.Println("Message channel closed")
					return
				}

				log.Printf("Received message: %s", string(msg.Body))

				// Process the message
				if err := handler.HandleChunkedUpload(ctx, msg.Body); err != nil {
					log.Printf("Error processing message: %v", err)
					// Nack the message WITHOUT requeue to avoid infinite loop
					// Failed messages should go to dead-letter queue or be logged for manual investigation
					if nackErr := msg.Nack(false, false); nackErr != nil {
						log.Printf("Failed to nack message: %v", nackErr)
					}
					continue
				}

				// Ack the message on success
				if err := msg.Ack(false); err != nil {
					log.Printf("Failed to ack message: %v", err)
				}
				log.Println("Message processed successfully")
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received")
	cancel()
	log.Println("Consumer service stopped gracefully")
}
