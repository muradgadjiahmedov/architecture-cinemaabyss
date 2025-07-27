package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	brokerAddr = getenv("KAFKA_BROKERS", "kafka:9092")
	topic      = getenv("KAFKA_TOPIC", "events")
	port       = getenv("PORT", "8082")
)

func main() {
	go consume() // запускаем консюмер параллельно

	http.HandleFunc("/api/events/health", health)
	http.HandleFunc("/api/events/movie", handler("movie"))
	http.HandleFunc("/api/events/user", handler("user"))
	http.HandleFunc("/api/events/payment", handler("payment"))

	log.Printf("events‑service on :%s  broker=%s  topic=%s\n", port, brokerAddr, topic)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// health‑чек
func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":true}`))
}

// factory‑функция для трёх эндпоинтов
func handler(eventType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		payload["type"] = eventType
		payload["ts"] = time.Now().UnixMilli()

		msg, _ := json.Marshal(payload)
		if err := produce(ctx, msg); err != nil {
			log.Println("kafka write err:", err)
			http.Error(w, "kafka error", http.StatusBadGateway)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status":"success"}`))
	}
}

// producer
func produce(ctx context.Context, val []byte) error {
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddr),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer w.Close()
	return w.WriteMessages(ctx, kafka.Message{Value: val})
}

// consumer (простой логгер)
func consume() {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr},
		GroupID:  "events-consumer",
		Topic:    topic,
		MinBytes: 1e3, MaxBytes: 1e6,
	})
	defer r.Close()

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Println("kafka read err:", err)
			time.Sleep(time.Second)
			continue
		}
		log.Printf("consumed: %s\n", string(m.Value))
	}
}

// утилити
func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
