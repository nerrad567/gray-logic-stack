package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Auth constants.
const (
	// ticketTTL is how long a WebSocket ticket is valid.
	ticketTTL = 60 * time.Second

	// devUsername is the hardcoded dev user (replaced in M1.5 with real user DB).
	devUsername = "admin"
	devPassword = "admin"
)

// loginRequest is the request body for POST /auth/login.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginResponse is the response body for POST /auth/login.
type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// ticketStore holds pending WebSocket authentication tickets.
// Tickets are single-use and expire after ticketTTL.
type ticketStore struct {
	tickets map[string]ticketEntry
	mu      sync.Mutex
}

type ticketEntry struct {
	expiresAt time.Time
}

var wsTickets = &ticketStore{
	tickets: make(map[string]ticketEntry),
}

// handleLogin authenticates a user and returns a JWT token.
// DEV ONLY: accepts admin/admin. Replace with real user DB in M1.5.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	// Dev-only credential check
	if req.Username != devUsername || req.Password != devPassword {
		writeUnauthorized(w, "invalid credentials")
		return
	}

	// Generate JWT
	ttl := s.secCfg.JWT.AccessTokenTTL
	if ttl == 0 {
		ttl = 15 // default 15 minutes
	}

	claims := jwt.MapClaims{
		"sub":  req.Username,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Duration(ttl) * time.Minute).Unix(),
		"role": "admin",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.secCfg.JWT.Secret))
	if err != nil {
		writeInternalError(w, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   ttl * 60, // seconds
	})
}

// handleWSTicket generates a single-use WebSocket authentication ticket.
// The client uses this ticket to authenticate the WebSocket connection
// without exposing the JWT in the URL.
func (s *Server) handleWSTicket(w http.ResponseWriter, _ *http.Request) {
	ticket := generateTicket()

	wsTickets.mu.Lock()
	wsTickets.tickets[ticket] = ticketEntry{
		expiresAt: time.Now().Add(ticketTTL),
	}
	wsTickets.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"ticket":     ticket,
		"expires_in": int(ticketTTL.Seconds()),
	})
}

// validateTicket checks if a ticket is valid and consumes it (single-use).
func validateTicket(ticket string) bool {
	wsTickets.mu.Lock()
	defer wsTickets.mu.Unlock()

	entry, ok := wsTickets.tickets[ticket]
	if !ok {
		return false
	}

	// Remove ticket (single-use)
	delete(wsTickets.tickets, ticket)

	// Check expiry
	return time.Now().Before(entry.expiresAt)
}

// ticketBytes is the number of random bytes used for WebSocket tickets.
const ticketBytes = 32

// generateTicket creates a cryptographically random ticket string.
func generateTicket() string {
	b := make([]byte, ticketBytes)
	//nolint:errcheck // crypto/rand.Read always returns len(b) on supported platforms
	rand.Read(b)
	return hex.EncodeToString(b)
}

// cleanExpiredTickets removes expired tickets from the store.
func cleanExpiredTickets() {
	wsTickets.mu.Lock()
	defer wsTickets.mu.Unlock()

	now := time.Now()
	for ticket, entry := range wsTickets.tickets {
		if now.After(entry.expiresAt) {
			delete(wsTickets.tickets, ticket)
		}
	}
}

// cleanTicketsLoop runs cleanExpiredTickets periodically until the context is cancelled.
func (s *Server) cleanTicketsLoop(ctx context.Context) {
	ticker := time.NewTicker(ticketTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanExpiredTickets()
		}
	}
}
