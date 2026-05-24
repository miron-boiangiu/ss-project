package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"mqtt-streaming-server/utils" // Encryption and decryption utilities
)

type ReportFunc func(ctx context.Context, db *pgxpool.Pool, params map[string]string) (interface{}, error)

// ReportRegistry este "inima" extensibilităţii.
// Când vrei să adaugi Raportul 6, doar îl adaugi în această listă!
var ReportRegistry = map[string]ReportFunc{
	"expirari":      GenerateExpirariReport,
	"anonimizate":   GenerateAnonimizateReport,
	"performanta":   GeneratePerformantaReport,
	"activitate_1m": GenerateActivitate1LunaReport,
	"statistici":    GenerateStatisticiGeneraleReport,
}

type ExpirareRecord struct {
	ID                 string `json:"id"`
	Nume               string `json:"nume"`
	Prenume            string `json:"prenume"`
	AvizMedical        string `json:"aviz_medical"`
	DataUrmatoare      string `json:"data_urm_examinari"`
}

// GenerateExpirariReport fetches medical records that are expiring within a certain timeframe
func GenerateExpirariReport(ctx context.Context, db *pgxpool.Pool, params map[string]string) (interface{}, error) {
	// 1. Define the timeframe window (default: expiring in the next 30 days)
	days := 30
	// TODO: Extract value from params["days"] if dynamic filtering from the frontend is required

	// 2. Query the database
	// Fetch rows where data_urm_examinari is populated.
	// Since data_urm_examinari is stored as TEXT, we fetch all relevant records and parse dates in Go.
	query := `
		SELECT id, nume, prenume, aviz_medical, data_urm_examinari 
		FROM photos 
		WHERE data_urm_examinari != '' 
		ORDER BY timestamp DESC
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()

	var expirari []ExpirareRecord
	
	now := time.Now()
	threshold := now.AddDate(0, 0, days)

	for rows.Next() {
		var id, encNume, encPrenume, aviz, dataUrm string
		
		if err := rows.Scan(&id, &encNume, &encPrenume, &aviz, &dataUrm); err != nil {
			continue // Skip malformed rows
		}

		// Parse data_urm_examinari (TEXT) into a valid time.Time object in Go
		parsedDate, err := time.Parse("02.01.2006", dataUrm)
		if err != nil {
			// Fallback in case the OCR parsed the date as YYYY-MM-DD
			parsedDate, err = time.Parse("2006-01-02", dataUrm)
			if err != nil {
				continue // Skip rows with unparseable dates
			}
		}

		// Check if the next examination date is in the past or within the next X days
		if parsedDate.Before(threshold) {
			// Decrypt sensitive fields using the application crypto utility
			decNume, _ := utils.Decrypt(encNume)
			decPrenume, _ := utils.Decrypt(encPrenume)

			// Append the decrypted record to the report slice
			expirari = append(expirari, ExpirareRecord{
				ID:            id,
				Nume:          decNume,
				Prenume:       decPrenume,
				AvizMedical:   aviz,
				DataUrmatoare: dataUrm,
			})
		}
	}

	// Handle empty slice scenario to ensure it returns an empty JSON array [] instead of null
	if expirari == nil {
		expirari = make([]ExpirareRecord, 0)
	}

	return expirari, nil
}

// AnonimizatRecord represents a research-safe dataset without Personal Health Information (PHI)
type AnonimizatRecord struct {
	ID              string    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	AvizMedical     string    `json:"aviz_medical"`
	ControlAngajare bool      `json:"control_angajare"`
	ControlPeriodic bool      `json:"control_periodic"`
	// We strictly exclude Name, CNP, and other identifying fields for compliance
}

// GenerateAnonimizateReport fetches a safe dataset for analytics, completely omitting PHI
func GenerateAnonimizateReport(ctx context.Context, db *pgxpool.Pool, params map[string]string) (interface{}, error) {
	// Query only non-sensitive columns useful for medical statistics
	query := `
		SELECT id, timestamp, aviz_medical, control_angajare, control_periodic 
		FROM photos 
		ORDER BY timestamp DESC
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query database for anonymized data: %w", err)
	}
	defer rows.Close()

	var dataset []AnonimizatRecord

	for rows.Next() {
		var rec AnonimizatRecord
		
		// Scan directly into our struct, as there is no decryption needed
		if err := rows.Scan(&rec.ID, &rec.Timestamp, &rec.AvizMedical, &rec.ControlAngajare, &rec.ControlPeriodic); err != nil {
			continue // Skip malformed rows
		}
		
		dataset = append(dataset, rec)
	}

	// Handle empty slice scenario
	if dataset == nil {
		dataset = make([]AnonimizatRecord, 0)
	}

	return dataset, nil
}
// PerformantaRecord holds system performance and throughput metrics
type PerformantaRecord struct {
	DeviceID    string `json:"device_id"`
	TotalPhotos int    `json:"total_photos"`
	// TODO: Uncomment these fields once the Confidence Thresholding task is merged and DB is updated
	// AvgLatencyMs   float64 `json:"average_latency_ms"`
	// AvgConfidence  float64 `json:"average_confidence"`
}

// GeneratePerformantaReport calculates system throughput and prepares for OCR latency metrics
func GeneratePerformantaReport(ctx context.Context, db *pgxpool.Pool, params map[string]string) (interface{}, error) {
	// Currently, we measure performance by the volume of successfully processed documents per device.
	// Once the database schema is updated with latency and confidence columns, 
	// this query can easily be expanded with AVG(ocr_latency_ms).
	query := `
		SELECT device_id, COUNT(*) as total_photos 
		FROM photos 
		GROUP BY device_id
		ORDER BY total_photos DESC
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query performance data: %w", err)
	}
	defer rows.Close()

	var stats []PerformantaRecord

	for rows.Next() {
		var rec PerformantaRecord
		
		if err := rows.Scan(&rec.DeviceID, &rec.TotalPhotos); err != nil {
			continue // Skip malformed rows
		}
		
		stats = append(stats, rec)
	}

	// Handle empty slice scenario
	if stats == nil {
		stats = make([]PerformantaRecord, 0)
	}

	return stats, nil
}
// RecentActivityReport represents the summary of medical checks in the last 30 days
type RecentActivityReport struct {
	TotalLastMonth  int            `json:"total_last_month"`
	StatusBreakdown map[string]int `json:"status_breakdown"`
}

// GenerateActivitate1LunaReport calculates how many checkups were done in the last 30 days
func GenerateActivitate1LunaReport(ctx context.Context, db *pgxpool.Pool, params map[string]string) (interface{}, error) {
	// Query to get a breakdown of medical opinions from the last 30 days
	query := `
		SELECT aviz_medical, COUNT(*) as count 
		FROM photos 
		WHERE timestamp >= NOW() - INTERVAL '30 days'
		GROUP BY aviz_medical
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent activity: %w", err)
	}
	defer rows.Close()

	breakdown := make(map[string]int)
	total := 0

	for rows.Next() {
		var aviz string
		var count int
		
		if err := rows.Scan(&aviz, &count); err != nil {
			continue // Skip malformed rows
		}
		
		// Map empty strings to "NEPROCESAT" or "PENDING" for clearer reporting
		if aviz == "" {
			aviz = "PENDING_REVIEW"
		}
		
		breakdown[aviz] = count
		total += count
	}

	// Prepare the final structured response
	report := RecentActivityReport{
		TotalLastMonth:  total,
		StatusBreakdown: breakdown,
	}

	return report, nil
}

// StatisticiGeneraleReport represents the all-time statistical overview of the medical data
type StatisticiGeneraleReport struct {
	TotalDocumente int `json:"total_documente"`
	Avize          struct {
		Apt            int `json:"apt"`
		AptConditionat int `json:"apt_conditionat"`
		InaptTemporar  int `json:"inapt_temporar"`
		Inapt          int `json:"inapt"`
		Neprocesat     int `json:"neprocesat"`
	} `json:"avize"`
	Controale struct {
		Angajare     int `json:"angajare"`
		Periodic     int `json:"periodic"`
		Adaptare     int `json:"adaptare"`
		Reluare      int `json:"reluare"`
		Supraveghere int `json:"supraveghere"`
		Alte         int `json:"alte"`
	} `json:"controale"`
}

// GenerateStatisticiGeneraleReport aggregates all-time system usage and medical outcomes
func GenerateStatisticiGeneraleReport(ctx context.Context, db *pgxpool.Pool, params map[string]string) (interface{}, error) {
	// We use conditional aggregation (FILTER WHERE) to get all statistics in a single, highly efficient database query
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE aviz_medical = 'APT') as apt,
			COUNT(*) FILTER (WHERE aviz_medical = 'APT CONDITIONAT') as apt_cond,
			COUNT(*) FILTER (WHERE aviz_medical = 'INAPT TEMPORAR') as inapt_temp,
			COUNT(*) FILTER (WHERE aviz_medical = 'INAPT') as inapt,
			COUNT(*) FILTER (WHERE aviz_medical = '') as neprocesat,
			COUNT(*) FILTER (WHERE control_angajare = true) as angajare,
			COUNT(*) FILTER (WHERE control_periodic = true) as periodic,
			COUNT(*) FILTER (WHERE control_adaptare = true) as adaptare,
			COUNT(*) FILTER (WHERE control_reluare = true) as reluare,
			COUNT(*) FILTER (WHERE control_supraveghere = true) as supraveghere,
			COUNT(*) FILTER (WHERE control_alte = true) as alte
		FROM photos
	`

	var stats StatisticiGeneraleReport

	// Use QueryRow since we are aggregating everything into exactly one result row
	err := db.QueryRow(ctx, query).Scan(
		&stats.TotalDocumente,
		&stats.Avize.Apt,
		&stats.Avize.AptConditionat,
		&stats.Avize.InaptTemporar,
		&stats.Avize.Inapt,
		&stats.Avize.Neprocesat,
		&stats.Controale.Angajare,
		&stats.Controale.Periodic,
		&stats.Controale.Adaptare,
		&stats.Controale.Reluare,
		&stats.Controale.Supraveghere,
		&stats.Controale.Alte,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to aggregate general statistics: %w", err)
	}

	return stats, nil
}