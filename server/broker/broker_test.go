package broker_test

import (
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jackc/pgx/v5/pgxpool"

	"mqtt-streaming-server/broker"
	"mqtt-streaming-server/utils"
)

func TestBrokerHandler_RegisterDevice(t *testing.T) {
	tests := []struct {
		name string
		db   *pgxpool.Pool
		ocr  utils.OCRRunner
		msg  mqtt.Message
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := broker.NewBrokerHandler(tt.db, tt.ocr, 80)
			b.RegisterDevice(nil, tt.msg)
		})
	}
}
