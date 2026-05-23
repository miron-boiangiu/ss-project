package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/otiai10/gosseract/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mqtt-streaming-server/broker"
	"mqtt-streaming-server/routes"
)

// TODO: Implement mTLS security
// See docs/SECURITY_IMPLEMENTATION.md for instructions on how to configure TLS

func main() {
	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := fmt.Sprintf("mongodb://%s:%s@mongo-db:27017/?authSource=admin", os.Getenv("MONGO_INITDB_ROOT_USERNAME"), os.Getenv("MONGO_INITDB_ROOT_PASSWORD"))
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		fmt.Println("Failed to connect to MongoDB:", err)
		panic(err)
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
	db := mongoClient.Database("mqtt-streaming-server")

	fmt.Println("Connected to MongoDB!")

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ocrClient := gosseract.NewClient()
	ocrClient.SetLanguage("eng", "ron")
	defer ocrClient.Close()
	brokerHandler := broker.NewBrokerHandler(db, ocrClient, readReviewThreshold())

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
	handler := routes.InitRoutes(db, client)

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
