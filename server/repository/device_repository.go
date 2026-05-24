package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"mqtt-streaming-server/domain"
)

type deviceRepository struct {
	db *pgxpool.Pool
}

func NewDeviceRepository(db *pgxpool.Pool) *deviceRepository {
	return &deviceRepository{db: db}
}

func (repo *deviceRepository) GetAllDevices(ctx context.Context) ([]*domain.Device, error) {
	rows, err := repo.db.Query(ctx,
		"SELECT id, device_id, device_name, device_status, ip_address, port, last_seen FROM devices")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	devices := make([]*domain.Device, 0)
	for rows.Next() {
		var device domain.Device
		err := rows.Scan(&device.ID, &device.DeviceID, &device.DeviceName,
			&device.DeviceStatus, &device.IPAddress, &device.Port, &device.LastSeen)
		if err != nil {
			return nil, err
		}
		devices = append(devices, &device)
	}
	return devices, rows.Err()
}

func (repo *deviceRepository) Save(ctx context.Context, device *domain.Device) error {
	_, err := repo.db.Exec(ctx,
		`INSERT INTO devices (device_id, device_name, device_status, ip_address, port, last_seen)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		device.DeviceID, device.DeviceName, device.DeviceStatus,
		device.IPAddress, device.Port, device.LastSeen)
	return err
}

func (repo *deviceRepository) Update(ctx context.Context, deviceID string, device *domain.Device) error {
	_, err := repo.db.Exec(ctx,
		`UPDATE devices SET device_name = $1, device_status = $2, ip_address = $3, port = $4, last_seen = $5
		 WHERE device_id = $6`,
		device.DeviceName, device.DeviceStatus, device.IPAddress, device.Port, device.LastSeen, deviceID)
	return err
}

func (repo *deviceRepository) GetByID(ctx context.Context, deviceID string) (*domain.Device, error) {
	var device domain.Device
	err := repo.db.QueryRow(ctx,
		"SELECT id, device_id, device_name, device_status, ip_address, port, last_seen FROM devices WHERE device_id = $1",
		deviceID).
		Scan(&device.ID, &device.DeviceID, &device.DeviceName,
			&device.DeviceStatus, &device.IPAddress, &device.Port, &device.LastSeen)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &device, nil
}
