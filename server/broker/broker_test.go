package broker_test

import (
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otiai10/gosseract/v2"

	"mqtt-streaming-server/broker"
)

func TestBrokerHandler_RegisterDevice(t *testing.T) {
	tests := []struct {
		name string
		db        *pgxpool.Pool
		ocrClient *gosseract.Client
		msg mqtt.Message
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := broker.NewBrokerHandler(tt.db, tt.ocrClient, 80)
			b.RegisterDevice(nil, tt.msg)
		})
	}
}
