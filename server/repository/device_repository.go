package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"

	"mqtt-streaming-server/domain"
)

type deviceRepository struct {
	db *mongo.Database
}

func NewDeviceRepository(db *mongo.Database) *deviceRepository {
	return &deviceRepository{db: db}
}

func (repo *deviceRepository) GetAllDevices(ctx context.Context) ([]*domain.Device, error) {
	collection := repo.db.Collection("devices")
	var devices []*domain.Device
	cursor, err := collection.Find(ctx, map[string]any{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var device domain.Device
		if err := cursor.Decode(&device); err != nil {
			return nil, err
		}
		devices = append(devices, &device)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

func (repo *deviceRepository) Save(ctx context.Context, device *domain.Device) error {
	collection := repo.db.Collection("devices")
	_, err := collection.InsertOne(ctx, device)
	return err
}

func (repo *deviceRepository) Update(ctx context.Context, deviceID string, device *domain.Device) error {
	collection := repo.db.Collection("devices")
	_, err := collection.UpdateOne(ctx, map[string]string{"device_id": deviceID}, map[string]any{"$set": device})
	return err
}

func (repo *deviceRepository) GetByID(ctx context.Context, deviceID string) (*domain.Device, error) {
	collection := repo.db.Collection("devices")
	var device *domain.Device
	err := collection.FindOne(ctx, map[string]string{"device_id": deviceID}).Decode(&device)
	if err != nil {
		return nil, err
	}
	return device, nil
}
