package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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
	brokerHandler := broker.NewBrokerHandler(pool, ocrClient)

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://broker:1883")
	opts.SetClientID("web")

	// Start the connection
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Subscribe to images topic
	if token := client.Subscribe("ssproject/images/#", 0, brokerHandler.HandlePhoto); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	if token := client.Subscribe("register/#", 0, brokerHandler.RegisterDevice); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	if token := client.Subscribe("device/id/#", 0, brokerHandler.DisconnectDevice); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	// Initialize user routes
	handler := routes.InitRoutes(pool, client)

	go func() {
		fmt.Println("Starting HTTP server on port 8080...")
		if err := http.ListenAndServe(":8080", handler); err != nil {
			panic(err)
		}
	}()

	<-c
}
