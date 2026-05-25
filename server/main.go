package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otiai10/gosseract/v2"

	"mqtt-streaming-server/broker"
	"mqtt-streaming-server/routes"
	"mqtt-streaming-server/utils"
)

func main() {
	// Initialize encryption
	if err := utils.InitEncryption(); err != nil {
		fmt.Println("Failed to initialize encryption:", err)
		panic(err)
	}

	// Connect to PostgreSQL
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connStr := fmt.Sprintf("postgres://%s:%s@postgres:5432/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"))

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Println("Failed to connect to PostgreSQL:", err)
		panic(err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		fmt.Println("Failed to ping PostgreSQL:", err)
		panic(err)
	}

	fmt.Println("Connected to PostgreSQL!")

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ocrClient := gosseract.NewClient()
	ocrClient.SetLanguage("eng", "ron")
	defer ocrClient.Close()
	brokerHandler := broker.NewBrokerHandler(pool, ocrClient, readReviewThreshold())

	tlsConfig := &tls.Config{
		ServerName: "broker",
		RootCAs:    x509.NewCertPool(),
	}
	caCert, err := os.ReadFile("/run/secrets/ca.crt")
	if err != nil {
		panic(fmt.Errorf("failed to read CA cert: %w", err))
	}
	if !tlsConfig.RootCAs.AppendCertsFromPEM(caCert) {
		panic("failed to parse CA certificate")
	}
	clientCert, err := tls.LoadX509KeyPair("/run/secrets/web.crt", "/run/secrets/web.key")
	if err != nil {
		panic(fmt.Errorf("failed to load client cert: %w", err))
	}
	tlsConfig.Certificates = []tls.Certificate{clientCert}

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tls://broker:8883")
	opts.SetClientID("web")
	opts.SetTLSConfig(tlsConfig)

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(30 * time.Second)

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("Connected to MQTT broker")

		subs := []struct {
			topic string
			cb    mqtt.MessageHandler
		}{
			{"ssproject/images/#", brokerHandler.HandlePhoto},
			{"register/#", brokerHandler.RegisterDevice},
			{"device/id/#", brokerHandler.DisconnectDevice},
		}
		for _, s := range subs {
			if token := c.Subscribe(s.topic, 0, s.cb); token.Wait() && token.Error() != nil {
				log.Printf("Subscribe error on %s: %v", s.topic, token.Error())
			}
		}
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	})

	opts.SetReconnectingHandler(func(c mqtt.Client, opts *mqtt.ClientOptions) {
		log.Println("Reconnecting to MQTT broker...")
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("Failed to connect to MQTT broker: %v", token.Error())
	} else {
		log.Println("Initial MQTT connection established")
	}

	handler := routes.InitRoutes(pool, client)

	go func() {
		fmt.Println("Starting HTTP server on port 8080...")
		if err := http.ListenAndServe(":8080", handler); err != nil {
			panic(err)
		}
	}()

	<-c
}

// readReviewThreshold returns the OCR confidence threshold below which an
// extracted document is flagged for manual review. It is configured via the
// OCR_REVIEW_THRESHOLD environment variable and defaults to 80.
func readReviewThreshold() float64 {
	const defaultThreshold = 80.0
	v := os.Getenv("OCR_REVIEW_THRESHOLD")
	if v == "" {
		return defaultThreshold
	}
	t, err := strconv.ParseFloat(v, 64)
	if err != nil {
		fmt.Printf("Invalid OCR_REVIEW_THRESHOLD %q, using default %.0f\n", v, defaultThreshold)
		return defaultThreshold
	}
	return t
}
