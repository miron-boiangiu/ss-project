// ocr-service is a minimal HTTP wrapper around libtesseract. It runs in an
// isolated, low-privilege container so that an exploit against the OCR engine
// (libtesseract / libleptonica / image decoders) is contained: the process
// holds no secrets, has no database access, and is on a Docker network with no
// egress to the internet.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/otiai10/gosseract/v2"
)

// WordBox mirrors the shape consumed by the API server's utils.WordBox so
// responses can be deserialized directly.
type WordBox struct {
	Word       string          `json:"word"`
	Confidence float64         `json:"confidence"`
	BBox       image.Rectangle `json:"bbox"`
	Offset     int             `json:"offset"`
}

type ocrResponse struct {
	Text  string    `json:"text"`
	Words []WordBox `json:"words"`
}

type service struct {
	mu       sync.Mutex
	client   *gosseract.Client
	maxBytes int64
}

func newService(languages []string, maxBytes int64) (*service, error) {
	c := gosseract.NewClient()
	if err := c.SetLanguage(languages...); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("set language: %w", err)
	}
	return &service{client: c, maxBytes: maxBytes}, nil
}

func (s *service) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

func (s *service) handleOCR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.maxBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "read failed", http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// Reject anything that isn't a parsable JPEG/PNG before handing bytes to
	// libtesseract. The sandbox is the real defense; this is just defense in
	// depth against trivially malformed inputs.
	if _, _, err := image.DecodeConfig(bytes.NewReader(body)); err != nil {
		http.Error(w, "invalid image format", http.StatusBadRequest)
		return
	}

	text, words, err := s.runOCR(body)
	if err != nil {
		log.Printf("ocr failed: %v", err)
		http.Error(w, "ocr failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ocrResponse{Text: text, Words: words})
}

func (s *service) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// runOCR is serialized — gosseract.Client wraps a single Tesseract API handle
// and is not safe for concurrent use.
func (s *service) runOCR(imageData []byte) (string, []WordBox, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.client.SetImageFromBytes(imageData); err != nil {
		return "", nil, fmt.Errorf("set image: %w", err)
	}
	boxes, err := s.client.GetBoundingBoxesVerbose()
	if err != nil {
		return "", nil, fmt.Errorf("bounding boxes: %w", err)
	}

	var sb strings.Builder
	words := make([]WordBox, 0, len(boxes))
	for i, box := range boxes {
		if i > 0 {
			prev := boxes[i-1]
			if box.BlockNum != prev.BlockNum || box.ParNum != prev.ParNum || box.LineNum != prev.LineNum {
				sb.WriteByte('\n')
			} else {
				sb.WriteByte(' ')
			}
		}
		words = append(words, WordBox{
			Word:       box.Word,
			Confidence: box.Confidence,
			BBox:       box.Box,
			Offset:     sb.Len(),
		})
		sb.WriteString(box.Word)
	}
	return sb.String(), words, nil
}

func main() {
	langs := splitCSV(getenv("OCR_LANGUAGES", "eng,ron"))
	maxBytes := parseInt64(getenv("OCR_MAX_BYTES", "26214400"), 26214400) // 25 MiB
	addr := getenv("OCR_LISTEN_ADDR", ":9090")

	svc, err := newService(langs, maxBytes)
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	defer svc.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/ocr", svc.handleOCR)
	mux.HandleFunc("/healthz", svc.handleHealth)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	log.Printf("ocr-service listening on %s (langs=%v, max_bytes=%d)", addr, langs, maxBytes)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server: %v", err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseInt64(s string, def int64) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}
