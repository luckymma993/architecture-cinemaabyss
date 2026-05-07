package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

var kafkaBroker = getEnv("KAFKA_BROKERS", "localhost:9092")

func main() {
	port := getEnv("PORT", "8082")

	go consumeEvents("movie-events")
	go consumeEvents("user-events")
	go consumeEvents("payment-events")

	http.HandleFunc("/api/events/health", handleHealth)
	http.HandleFunc("/api/events/movie", handleEvent("movie-events"))
	http.HandleFunc("/api/events/user", handleEvent("user-events"))
	http.HandleFunc("/api/events/payment", handleEvent("payment-events"))

	log.Printf("Starting events service on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server startup error: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": true}`))
}

func handleEvent(topic string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, `{"error": "Empty request body"}`, http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		err = produceEvent(topic, body)
		if err != nil {
			log.Printf("Error writing to Kafka (%s): %v", topic, err)
			http.Error(w, `{"error": "Failed to save event"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		
		response := fmt.Sprintf(`{"status": "success", "partition": 0, "offset": 0, "event": {"id": "generated-id", "type": "%s", "timestamp": "%s", "payload": %s}}`, 
			topic, time.Now().Format(time.RFC3339), string(body))
		w.Write([]byte(response))
	}
}

func produceEvent(topic string, message []byte) error {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	msg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", time.Now().UnixNano())),
		Value: message,
	}

	return writer.WriteMessages(context.Background(), msg)
}

func consumeEvents(topic string) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaBroker},
		Topic:     topic,
		Partition: 0,
		MinBytes:  10e3,
		MaxBytes:  10e6,
	})
	defer reader.Close()

	log.Printf("Started listening to topic: %s", topic)

	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading from Kafka (%s): %v", topic, err)
			time.Sleep(time.Second * 5)
			continue
		}
		
		var prettyJSON map[string]interface{}
		json.Unmarshal(m.Value, &prettyJSON)
		
		log.Printf("[KAFKA CONSUMER | Topic: %s] Event received: %v", topic, prettyJSON)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
