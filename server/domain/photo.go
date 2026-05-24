package domain

import (
	"context"
	"time"
)

type Photo struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	ImageType    string    `json:"image_type"`
	PresignedURL string    `json:"presigned_url,omitempty"`
	DeviceID     string    `json:"device_id"`
	Text         string    `json:"text"`

	// Medical Data Fields
	UnitateMedicala        string `json:"unitate_medicala"`
	AdresaUnitateMedicala  string `json:"adresa_unitate_medicala"`
	TelefonUnitateMedicala string `json:"telefon_unitate_medicala"`
	NumarFisa              string `json:"numar_fisa"`
	SocietateUnitate       string `json:"societate_unitate"`
	AdresaAngajator        string `json:"adresa_angajator"`
	TelefonAngajator       string `json:"telefon_angajator"`
	Nume                   string `json:"nume"`
	Prenume                string `json:"prenume"`
	CNP                    string `json:"cnp"`
	ProfesieFunctie        string `json:"profesie_functie"`
	LocDeMunca             string `json:"loc_de_munca"`
	TipControl             string `json:"tip_control"`
	ControlAngajare        bool   `json:"control_angajare"`
	ControlPeriodic        bool   `json:"control_periodic"`
	ControlAdaptare        bool   `json:"control_adaptare"`
	ControlReluare         bool   `json:"control_reluare"`
	ControlSupraveghere    bool   `json:"control_supraveghere"`
	ControlAlte            bool   `json:"control_alte"`

	AvizMedical string `json:"aviz_medical"`
	Recomandari        string `json:"recomandari"`
	Data               string `json:"data"`
	DataUrmExaminari   string `json:"data_urm_examinari"`

	// Confidence & review (PR 1, task 2.B)
	NeedsReview       bool               `json:"needs_review"`
	OverallConfidence float64            `json:"overall_confidence"`
	FieldConfidences  map[string]float64 `json:"field_confidences"`

	// Schema identification (PR 2, task 2.B)
	DocumentType  string `json:"document_type"`
	SchemaVersion string `json:"schema_version"`
}

type PhotoRepository interface {
	GetPhotos(ctx context.Context, filters map[string]any) ([]*Photo, error)
	GetByID(ctx context.Context, id string) (*Photo, error)
	Save(ctx context.Context, photo *Photo) error
	Delete(ctx context.Context, id string) error
	DeleteAll(ctx context.Context) (int64, error)
}
