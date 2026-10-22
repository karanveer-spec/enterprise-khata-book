package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"khata-book/models"
	"khata-book/services"
)

type ExportHandler struct {
	svc *services.KhataService
}

func NewExportHandler(svc *services.KhataService) *ExportHandler {
	return &ExportHandler{svc: svc}
}

// HandleExportCSV handles GET /api/nodes/{id}/export
// Streams the filtered entry list directly to a downloadable .csv file.
func (h *ExportHandler) HandleExportCSV(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	nodeID := r.PathValue("id")
	query := r.URL.Query()

	// Parse filtering options (no pagination limit for full export)
	filter := models.EntryFilter{
		NodeID:             nodeID,
		IncludeDescendants: query.Get("include_descendants") != "false", // default true for export
		Search:             strings.TrimSpace(query.Get("q")),
		Period:             strings.TrimSpace(query.Get("period")),
		FromDate:           strings.TrimSpace(query.Get("from_date")),
		ToDate:             strings.TrimSpace(query.Get("to_date")),
		Type:               strings.ToUpper(strings.TrimSpace(query.Get("type"))),
		SortBy:             "date",
		SortDir:            "desc",
		Page:               1,
		Limit:              10000, // Export up to 10,000 entries
	}

	res, err := h.svc.ListEntries(user, filter)
	if err != nil {
		HandleServiceError(w, err)
		return
	}

	fileName := fmt.Sprintf("Khata_Ledger_%s_%s.csv", nodeID, time.Now().Format("20060102_150405"))

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write CSV Header
	header := []string{
		"Transaction ID",
		"Date (YYYY-MM-DD)",
		"Ledger Node",
		"Type",
		"Reference / Voucher",
		"Description",
		"Amount in Paise",
		"Amount in Rupees (₹)",
		"Formatted Amount",
	}
	if err := csvWriter.Write(header); err != nil {
		return
	}

	// Stream entries
	for _, e := range res.Items {
		typeLabel := "Jama (Credit)"
		if e.Type == models.EntryTypeDebit {
			typeLabel = "Udhar (Debit)"
		}

		row := []string{
			e.ID,
			e.EntryDate,
			e.NodeName,
			typeLabel,
			e.Reference,
			e.Description,
			fmt.Sprintf("%d", e.AmountPaise),
			fmt.Sprintf("%.2f", e.AmountRupees),
			e.FormattedAmount,
		}
		if err := csvWriter.Write(row); err != nil {
			return
		}
	}
}
