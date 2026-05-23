package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v4"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jackc/pgx/v5/pgxpool"
)

func InitRoutes(db *pgxpool.Pool, mqttClient mqtt.Client) http.Handler {
	mux := http.NewServeMux()
	InitUserRoutes(db, mux)
	InitPhotoRoutes(db, mux)
	InitDeviceRoutes(db, mqttClient, mux)

    mux.Handle("/api/reports", withAuth(http.HandlerFunc(GenerateReportHandler(db))))
	// Serve static files from ./uploads
	fs := http.FileServer(http.Dir("uploads"))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

	// Broker info endpoint - returns the MQTT broker connection info
	mux.HandleFunc("/broker-info", handleBrokerInfo)

	corsHandler := withCORS(mux)

	return corsHandler
}

func handleBrokerInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := getOutboundIP()
	port := "1883"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"ip":   ip,
		"port": port,
	})
}

func getOutboundIP() string {
	if hostIP := os.Getenv("MQTT_HOST_IP"); hostIP != "" {
		if addrs, err := net.LookupHost(hostIP); err == nil && len(addrs) > 0 {
			return addrs[0]
		}
		if net.ParseIP(hostIP) != nil {
			return hostIP
		}
	}

	addrs, err := net.LookupHost("host.docker.internal")
	if err == nil && len(addrs) > 0 {
		return addrs[0]
	}

	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := authHeader[7:]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		email, _ := claims["email"].(string)
		role, _ := claims["role"].(string)

		ctx := context.WithValue(r.Context(), "email", email)
		ctx = context.WithValue(ctx, "role", role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
