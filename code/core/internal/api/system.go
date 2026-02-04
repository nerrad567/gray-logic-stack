package api

import (
	"encoding/json"
	"net/http"
)

// FactoryResetRequest defines the options for a factory reset.
type FactoryResetRequest struct {
	ClearDevices   bool   `json:"clear_devices"`
	ClearScenes    bool   `json:"clear_scenes"`
	ClearLocations bool   `json:"clear_locations"`
	ClearDiscovery bool   `json:"clear_discovery"`
	ClearSite      bool   `json:"clear_site"`
	Confirm        string `json:"confirm"`
}

// FactoryResetResponse reports what was deleted.
type FactoryResetResponse struct {
	Status  string         `json:"status"`
	Deleted map[string]int `json:"deleted"`
}

// handleFactoryReset clears selected data from the database in a single
// transaction, then refreshes all in-memory caches and bridge mappings.
//
// This is a destructive operation — the request must include an exact
// confirmation string as a safety guard.
func (s *Server) handleFactoryReset(w http.ResponseWriter, r *http.Request) {
	var req FactoryResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}

	// Safety guard: require exact confirmation string.
	if req.Confirm != "FACTORY RESET" {
		writeBadRequest(w, `confirm field must be exactly "FACTORY RESET"`)
		return
	}

	// Must select at least one category.
	if !req.ClearDevices && !req.ClearScenes && !req.ClearLocations && !req.ClearDiscovery && !req.ClearSite {
		writeBadRequest(w, "at least one clear_* option must be true")
		return
	}

	ctx := r.Context()
	db := s.db.SqlDB()
	deleted := make(map[string]int)

	// Execute all DELETEs in a single transaction, respecting FK order.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("factory reset: failed to begin transaction", "error", err)
		writeInternalError(w, "failed to begin transaction")
		return
	}
	defer tx.Rollback() //nolint:errcheck // rollback is a no-op after commit

	// Helper to execute a DELETE and record the count.
	deleteFrom := func(table string) error {
		result, err := tx.ExecContext(ctx, "DELETE FROM "+table)
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		deleted[table] = int(n)
		return nil
	}

	// Scenes (child table first).
	if req.ClearScenes {
		if err := deleteFrom("scene_executions"); err != nil {
			s.logger.Error("factory reset: failed to clear scene_executions", "error", err)
			writeInternalError(w, "failed to clear scene executions")
			return
		}
		if err := deleteFrom("scenes"); err != nil {
			s.logger.Error("factory reset: failed to clear scenes", "error", err)
			writeInternalError(w, "failed to clear scenes")
			return
		}
	}

	// Devices.
	if req.ClearDevices {
		if err := deleteFrom("devices"); err != nil {
			s.logger.Error("factory reset: failed to clear devices", "error", err)
			writeInternalError(w, "failed to clear devices")
			return
		}
	}

	// Locations (rooms before areas due to FK).
	if req.ClearLocations {
		if err := deleteFrom("rooms"); err != nil {
			s.logger.Error("factory reset: failed to clear rooms", "error", err)
			writeInternalError(w, "failed to clear rooms")
			return
		}
		if err := deleteFrom("areas"); err != nil {
			s.logger.Error("factory reset: failed to clear areas", "error", err)
			writeInternalError(w, "failed to clear areas")
			return
		}
	}

	// Discovery data.
	if req.ClearDiscovery {
		if err := deleteFrom("knx_group_addresses"); err != nil {
			s.logger.Error("factory reset: failed to clear knx_group_addresses", "error", err)
			writeInternalError(w, "failed to clear discovery data")
			return
		}
		if err := deleteFrom("knx_devices"); err != nil {
			s.logger.Error("factory reset: failed to clear knx_devices", "error", err)
			writeInternalError(w, "failed to clear discovery data")
			return
		}
	}

	// Site record (must come after areas due to FK).
	if req.ClearSite {
		// Areas must already be cleared (or not present) for this to succeed.
		// If clear_locations wasn't set, areas may still reference the site.
		if !req.ClearLocations {
			// Clear locations too — can't delete site with areas referencing it.
			if _, ok := deleted["rooms"]; !ok {
				if err := deleteFrom("rooms"); err != nil {
					s.logger.Error("factory reset: failed to clear rooms (for site)", "error", err)
					writeInternalError(w, "failed to clear rooms")
					return
				}
			}
			if _, ok := deleted["areas"]; !ok {
				if err := deleteFrom("areas"); err != nil {
					s.logger.Error("factory reset: failed to clear areas (for site)", "error", err)
					writeInternalError(w, "failed to clear areas")
					return
				}
			}
		}
		if err := deleteFrom("sites"); err != nil {
			s.logger.Error("factory reset: failed to clear sites", "error", err)
			writeInternalError(w, "failed to clear site record")
			return
		}
	}

	// Commit the transaction.
	if err := tx.Commit(); err != nil {
		s.logger.Error("factory reset: failed to commit transaction", "error", err)
		writeInternalError(w, "failed to commit factory reset")
		return
	}

	s.logger.Info("factory reset committed", "deleted", deleted)

	// Refresh in-memory caches after successful DB wipe.
	if req.ClearDevices || req.ClearScenes {
		if err := s.registry.RefreshCache(ctx); err != nil {
			s.logger.Warn("factory reset: failed to refresh device cache", "error", err)
		}
	}
	if req.ClearScenes && s.sceneRegistry != nil {
		if err := s.sceneRegistry.RefreshCache(ctx); err != nil {
			s.logger.Warn("factory reset: failed to refresh scene cache", "error", err)
		}
	}
	if s.knxBridge != nil {
		s.knxBridge.ReloadDevices(ctx)
		s.logger.Info("KNX bridge devices reloaded after factory reset")
	}

	writeJSON(w, http.StatusOK, FactoryResetResponse{
		Status:  "ok",
		Deleted: deleted,
	})
}
