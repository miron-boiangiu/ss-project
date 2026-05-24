package routes

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	// ATENȚIE: Înlocuiește "server/repository" cu calea corectă din proiectul tău (ex: "ss-project/server/repository")
	"mqtt-streaming-server/repository"
)

// GenerateReportHandler returnează funcția care procesează endpoint-ul HTTP
func GenerateReportHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Aflăm ce raport cere utilizatorul (din query param ?type=...)
		reportType := r.URL.Query().Get("type")
		if reportType == "" {
			http.Error(w, `{"error": "Lipseste parametrul 'type'"}`, http.StatusBadRequest)
			return
		}

		// 2. Căutăm raportul în Registrul nostru extensibil
		generateFunc, exists := repository.ReportRegistry[reportType]
		if !exists {
			http.Error(w, `{"error": "Tipul de raport solicitat nu exista in sistem"}`, http.StatusNotFound)
			return
		}

		// 3. Extragem toți parametrii opționali (ex: startDate, endDate) pentru a-i trimite la DB
		params := make(map[string]string)
		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}

		// 4. Executăm generarea efectivă a raportului
		data, err := generateFunc(r.Context(), db, params)
		if err != nil {
			http.Error(w, `{"error": "Eroare la generarea raportului: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// 5. Trimitem răspunsul cu succes în format JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}