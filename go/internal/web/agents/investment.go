package agents

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/wade/aia/internal/domain/investment"
	invService "github.com/wade/aia/internal/service/investment"
	"github.com/wade/aia/internal/web/host"
)

type InvestmentWebModule struct {
	svc *invService.Service
}

func NewInvestmentWebModule(svc *invService.Service) *InvestmentWebModule {
	return &InvestmentWebModule{svc: svc}
}

func (m *InvestmentWebModule) AgentID() string {
	return "investment"
}

func (m *InvestmentWebModule) NavItems() []host.NavItem {
	return []host.NavItem{
		{
			ID:       "investment",
			Label:    "Investment Portfolio",
			Path:     "#/investment",
			Icon:     "trending-up",
			Order:    10,
			Required: "investment.read",
		},
	}
}

// RegisterRoutes registers endpoints onto the isolated ServeMux dedicated to Investment Agent.
func (m *InvestmentWebModule) RegisterRoutes(mux *http.ServeMux) error {
	mux.HandleFunc("/profile", m.handleProfile)
	mux.HandleFunc("/snapshot", m.handleSnapshot)
	mux.HandleFunc("/instruments", m.handleInstruments)
	mux.HandleFunc("/decisions", m.handleDecisions)
	mux.HandleFunc("/proposals", m.handleProposals)
	mux.HandleFunc("/evaluate", m.handleEvaluateDecision)
	mux.HandleFunc("/proposals/", m.handleProposalActions)
	mux.HandleFunc("/sync", m.handleSync)
	mux.HandleFunc("/documents", m.handleDocuments)
	mux.HandleFunc("/market-data", m.handleMarketData)
	mux.HandleFunc("/predictions", m.handlePredictions)
	mux.HandleFunc("/predictions/evaluate", m.handleEvaluatePredictions)
	mux.HandleFunc("/evaluations", m.handleEvaluationResults)
	return nil
}

func (m *InvestmentWebModule) handleProfile(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ctx := r.Context()

	prof, err := m.svc.GetProfile(ctx)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	constraints, _ := m.svc.ListConstraints(ctx)

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"profile":     prof,
		"constraints": constraints,
	}, reqID)
}

func (m *InvestmentWebModule) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ctx := r.Context()

	snap, positions, err := m.svc.GetSnapshot(ctx)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"snapshot":  snap,
		"positions": positions,
	}, reqID)
}

func (m *InvestmentWebModule) handleInstruments(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	insts, err := m.svc.ListInstruments(r.Context())
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, insts, reqID)
}

func (m *InvestmentWebModule) handleDecisions(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ctx := r.Context()

	limit := 30
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}

	decisions, err := m.svc.ListDecisions(ctx, limit)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	rejections, _ := m.svc.ListRejections(ctx, limit)

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"decisions":  decisions,
		"rejections": rejections,
	}, reqID)
}

func (m *InvestmentWebModule) handleProposals(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	status := investment.ProposalStatus(r.URL.Query().Get("status"))
	limit := 30
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}

	proposals, err := m.svc.ListProposals(r.Context(), status, limit)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, proposals, reqID)
}

func (m *InvestmentWebModule) handleEvaluateDecision(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "POST required", reqID)
		return
	}

	var req struct {
		investment.AgyDecisionInput
		Price float64 `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		host.WriteError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error(), reqID)
		return
	}
	if req.Price <= 0 {
		host.WriteError(w, http.StatusBadRequest, "INVALID_PRICE", "Market price must be > 0", reqID)
		return
	}

	proposal, err := m.svc.ProcessAgyDecision(r.Context(), req.AgyDecisionInput, req.Price)
	if err != nil {
		host.WriteError(w, http.StatusUnprocessableEntity, "DECISION_REJECTED", err.Error(), reqID)
		return
	}

	host.WriteJSON(w, http.StatusCreated, map[string]any{
		"evaluated": true,
		"proposal":  proposal,
	}, reqID)
}

func (m *InvestmentWebModule) handleProposalActions(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	ctx := r.Context()

	// Path: /proposals/{id}/{approve|reject}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/proposals/"), "/"), "/")
	if len(parts) < 2 {
		host.WriteError(w, http.StatusBadRequest, "INVALID_PATH", "Format: /proposals/{id}/{action}", reqID)
		return
	}
	propID, action := parts[0], parts[1]

	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "POST required", reqID)
		return
	}

	switch action {
	case "approve":
		if err := m.svc.ApproveProposal(ctx, propID); err != nil {
			host.WriteError(w, http.StatusBadRequest, "APPROVE_FAILED", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "APPROVED"}, reqID)

	case "reject":
		var req struct {
			Reason string `json:"reason"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if err := m.svc.RejectProposal(ctx, propID, req.Reason); err != nil {
			host.WriteError(w, http.StatusBadRequest, "REJECT_FAILED", err.Error(), reqID)
			return
		}
		host.WriteJSON(w, http.StatusOK, map[string]string{"status": "REJECTED"}, reqID)

	default:
		host.WriteError(w, http.StatusNotFound, "UNKNOWN_ACTION", "Action must be approve or reject", reqID)
	}
}

func (m *InvestmentWebModule) handleSync(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "POST required", reqID)
		return
	}

	var req struct {
		Symbol string `json:"symbol"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Symbol == "" {
		req.Symbol = "2330" // default TSMC
	}

	ctx := r.Context()
	md, err := m.svc.SyncMarketData(ctx, req.Symbol)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "SYNC_FAILED", err.Error(), reqID)
		return
	}
	docs, _ := m.svc.SyncAnnouncements(ctx, req.Symbol)

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"synced":        true,
		"market_data":   md,
		"documents_cnt": len(docs),
	}, reqID)
}

func (m *InvestmentWebModule) handleDocuments(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	symbol := r.URL.Query().Get("symbol")
	limit := 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}

	docs, err := m.svc.ListRawDocuments(r.Context(), symbol, limit)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, docs, reqID)
}

func (m *InvestmentWebModule) handleMarketData(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		symbol = "2330"
	}
	limit := 30
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}

	bars, err := m.svc.ListHistoricalData(r.Context(), symbol, limit)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, bars, reqID)
}

func (m *InvestmentWebModule) handlePredictions(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	status := investment.PredictionStatus(r.URL.Query().Get("status"))
	limit := 30
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}

	preds, err := m.svc.ListPredictions(r.Context(), status, limit)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, preds, reqID)
}

func (m *InvestmentWebModule) handleEvaluatePredictions(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	if r.Method != http.MethodPost {
		host.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "POST required", reqID)
		return
	}

	outcomes, evalRes, err := m.svc.EvaluatePredictions(r.Context())
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "EVAL_FAILED", err.Error(), reqID)
		return
	}

	host.WriteJSON(w, http.StatusOK, map[string]any{
		"evaluated_outcomes": outcomes,
		"calibration_result": evalRes,
	}, reqID)
}

func (m *InvestmentWebModule) handleEvaluationResults(w http.ResponseWriter, r *http.Request) {
	reqID := r.Header.Get("X-Request-ID")
	limit := 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}

	results, err := m.svc.ListEvaluationResults(r.Context(), limit)
	if err != nil {
		host.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), reqID)
		return
	}
	host.WriteJSON(w, http.StatusOK, results, reqID)
}
