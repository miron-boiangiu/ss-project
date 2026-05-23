package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/utils"
)

type photoRepository struct {
	db *pgxpool.Pool
}

func NewPhotoRepository(db *pgxpool.Pool) *photoRepository {
	return &photoRepository{db: db}
}

func (repo *photoRepository) GetPhotos(ctx context.Context, filters map[string]any) ([]*domain.Photo, error) {
	query := `SELECT id, timestamp, image_type, device_id, text,
		unitate_medicala, adresa_unitate_medicala, telefon_unitate_medicala, numar_fisa,
		societate_unitate, adresa_angajator, telefon_angajator,
		nume, prenume, cnp, profesie_functie, loc_de_munca, tip_control,
		control_angajare, control_periodic, control_adaptare, control_reluare, control_supraveghere, control_alte,
		aviz_medical, recomandari, data, data_urm_examinari,
		needs_review, overall_confidence, field_confidences
		FROM photos`

	var conditions []string
	var args []any
	argIdx := 1

	if v, ok := filters["timestamp_gte"]; ok {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, v)
		argIdx++
	}
	if v, ok := filters["timestamp_lte"]; ok {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIdx))
		args = append(args, v)
		argIdx++
	}
	if v, ok := filters["text"]; ok {
		conditions = append(conditions, fmt.Sprintf("text ILIKE '%%' || $%d || '%%'", argIdx))
		args = append(args, v)
		argIdx++
	}
	if v, ok := filters["device_id"]; ok {
		conditions = append(conditions, fmt.Sprintf("device_id = $%d", argIdx))
		args = append(args, v)
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, cond := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	query += " ORDER BY timestamp DESC"

	rows, err := repo.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	photos := make([]*domain.Photo, 0)
	for rows.Next() {
		var photo domain.Photo
		var fieldConfidencesRaw []byte
		err := rows.Scan(
			&photo.ID, &photo.Timestamp, &photo.ImageType, &photo.DeviceID, &photo.Text,
			&photo.UnitateMedicala, &photo.AdresaUnitateMedicala, &photo.TelefonUnitateMedicala, &photo.NumarFisa,
			&photo.SocietateUnitate, &photo.AdresaAngajator, &photo.TelefonAngajator,
			&photo.Nume, &photo.Prenume, &photo.CNP, &photo.ProfesieFunctie, &photo.LocDeMunca, &photo.TipControl,
			&photo.ControlAngajare, &photo.ControlPeriodic, &photo.ControlAdaptare, &photo.ControlReluare, &photo.ControlSupraveghere, &photo.ControlAlte,
			&photo.AvizMedical, &photo.Recomandari, &photo.Data, &photo.DataUrmExaminari,
			&photo.NeedsReview, &photo.OverallConfidence, &fieldConfidencesRaw,
		)
		if err != nil {
			return nil, err
		}
		if err := unmarshalFieldConfidences(fieldConfidencesRaw, &photo); err != nil {
			return nil, err
		}
		if err := decryptPhotoFields(&photo); err != nil {
			return nil, err
		}
		photos = append(photos, &photo)
	}
	return photos, rows.Err()
}

func (repo *photoRepository) Save(ctx context.Context, photo *domain.Photo) error {
	if photo.ID == "" {
		photo.ID = uuid.New().String()
	}
	if photo.Timestamp.IsZero() {
		photo.Timestamp = time.Now().UTC()
	}

	if err := encryptPhotoFields(photo); err != nil {
		return err
	}

	fieldConfidencesJSON, err := marshalFieldConfidences(photo.FieldConfidences)
	if err != nil {
		return err
	}

	_, err = repo.db.Exec(ctx,
		`INSERT INTO photos (id, timestamp, image_type, device_id, text,
			unitate_medicala, adresa_unitate_medicala, telefon_unitate_medicala, numar_fisa,
			societate_unitate, adresa_angajator, telefon_angajator,
			nume, prenume, cnp, profesie_functie, loc_de_munca, tip_control,
			control_angajare, control_periodic, control_adaptare, control_reluare, control_supraveghere, control_alte,
			aviz_medical, recomandari, data, data_urm_examinari,
			needs_review, overall_confidence, field_confidences)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31)`,
		photo.ID, photo.Timestamp, photo.ImageType, photo.DeviceID, photo.Text,
		photo.UnitateMedicala, photo.AdresaUnitateMedicala, photo.TelefonUnitateMedicala, photo.NumarFisa,
		photo.SocietateUnitate, photo.AdresaAngajator, photo.TelefonAngajator,
		photo.Nume, photo.Prenume, photo.CNP, photo.ProfesieFunctie, photo.LocDeMunca, photo.TipControl,
		photo.ControlAngajare, photo.ControlPeriodic, photo.ControlAdaptare, photo.ControlReluare, photo.ControlSupraveghere, photo.ControlAlte,
		photo.AvizMedical, photo.Recomandari, photo.Data, photo.DataUrmExaminari,
		photo.NeedsReview, photo.OverallConfidence, fieldConfidencesJSON)
	return err
}

func (repo *photoRepository) GetByID(ctx context.Context, id string) (*domain.Photo, error) {
	var photo domain.Photo
	var fieldConfidencesRaw []byte
	err := repo.db.QueryRow(ctx,
		`SELECT id, timestamp, image_type, device_id, text,
			unitate_medicala, adresa_unitate_medicala, telefon_unitate_medicala, numar_fisa,
			societate_unitate, adresa_angajator, telefon_angajator,
			nume, prenume, cnp, profesie_functie, loc_de_munca, tip_control,
			control_angajare, control_periodic, control_adaptare, control_reluare, control_supraveghere, control_alte,
			aviz_medical, recomandari, data, data_urm_examinari,
			needs_review, overall_confidence, field_confidences
		 FROM photos WHERE id = $1`, id).
		Scan(
			&photo.ID, &photo.Timestamp, &photo.ImageType, &photo.DeviceID, &photo.Text,
			&photo.UnitateMedicala, &photo.AdresaUnitateMedicala, &photo.TelefonUnitateMedicala, &photo.NumarFisa,
			&photo.SocietateUnitate, &photo.AdresaAngajator, &photo.TelefonAngajator,
			&photo.Nume, &photo.Prenume, &photo.CNP, &photo.ProfesieFunctie, &photo.LocDeMunca, &photo.TipControl,
			&photo.ControlAngajare, &photo.ControlPeriodic, &photo.ControlAdaptare, &photo.ControlReluare, &photo.ControlSupraveghere, &photo.ControlAlte,
			&photo.AvizMedical, &photo.Recomandari, &photo.Data, &photo.DataUrmExaminari,
			&photo.NeedsReview, &photo.OverallConfidence, &fieldConfidencesRaw,
		)
	if err != nil {
		return nil, err
	}
	if err := unmarshalFieldConfidences(fieldConfidencesRaw, &photo); err != nil {
		return nil, err
	}
	if err := decryptPhotoFields(&photo); err != nil {
		return nil, err
	}
	return &photo, nil
}

func (repo *photoRepository) Delete(ctx context.Context, id string) error {
	_, err := repo.db.Exec(ctx, "DELETE FROM photos WHERE id = $1", id)
	return err
}

func (repo *photoRepository) DeleteAll(ctx context.Context) (int64, error) {
	result, err := repo.db.Exec(ctx, "DELETE FROM photos")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func encryptPhotoFields(photo *domain.Photo) error {
	fields := []*string{
		&photo.CNP, &photo.Nume, &photo.Prenume,
		&photo.ProfesieFunctie, &photo.LocDeMunca, &photo.NumarFisa,
		&photo.Recomandari, &photo.UnitateMedicala, &photo.SocietateUnitate,
		&photo.AdresaUnitateMedicala, &photo.AdresaAngajator,
		&photo.TelefonUnitateMedicala, &photo.TelefonAngajator,
	}
	for _, field := range fields {
		encrypted, err := utils.Encrypt(*field)
		if err != nil {
			return err
		}
		*field = encrypted
	}
	return nil
}

// marshalFieldConfidences serialises the map for the JSONB column. A nil map
// becomes "{}" rather than JSON null so the column never holds null payloads.
func marshalFieldConfidences(m map[string]float64) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

// unmarshalFieldConfidences reads the JSONB payload back onto the photo. Empty
// or "null" bytes leave FieldConfidences as nil so callers don't have to
// distinguish between absent and explicitly-null payloads.
func unmarshalFieldConfidences(raw []byte, photo *domain.Photo) error {
	if len(raw) == 0 || string(raw) == "null" {
		photo.FieldConfidences = nil
		return nil
	}
	return json.Unmarshal(raw, &photo.FieldConfidences)
}

func decryptPhotoFields(photo *domain.Photo) error {
	fields := []*string{
		&photo.CNP, &photo.Nume, &photo.Prenume,
		&photo.ProfesieFunctie, &photo.LocDeMunca, &photo.NumarFisa,
		&photo.Recomandari, &photo.UnitateMedicala, &photo.SocietateUnitate,
		&photo.AdresaUnitateMedicala, &photo.AdresaAngajator,
		&photo.TelefonUnitateMedicala, &photo.TelefonAngajator,
	}
	for _, field := range fields {
		decrypted, err := utils.Decrypt(*field)
		if err != nil {
			return err
		}
		*field = decrypted
	}
	return nil
}
