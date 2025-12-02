package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/backend/common"
	v1 "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/service"
	"github.com/gorilla/mux"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// SensitiveApprovalAPI handles HTTP requests for sensitive approval endpoints.
type SensitiveApprovalAPI struct {
	sensitiveApprovalService v1.SensitiveApprovalServiceClient
	sensitiveDataService     service.SensitiveDataService
	changeInterceptorService service.ChangeInterceptorService
	approvalService          service.ApprovalService
}

// NewSensitiveApprovalAPI creates a new SensitiveApprovalAPI.
func NewSensitiveApprovalAPI(
	sensitiveApprovalService v1.SensitiveApprovalServiceClient,
	sensitiveDataService service.SensitiveDataService,
	changeInterceptorService service.ChangeInterceptorService,
	approvalService service.ApprovalService,
) *SensitiveApprovalAPI {
	return &SensitiveApprovalAPI{
		sensitiveApprovalService: sensitiveApprovalService,
		sensitiveDataService:     sensitiveDataService,
		changeInterceptorService: changeInterceptorService,
		approvalService:          approvalService,
	}
}

// RegisterRoutes registers the sensitive approval API routes.
func (api *SensitiveApprovalAPI) RegisterRoutes(router *mux.Router) {
	// Sensitive Level endpoints
	router.HandleFunc("/api/v1/sensitive-levels", api.listSensitiveLevels).Methods("GET")
	router.HandleFunc("/api/v1/sensitive-levels/{id}", api.getSensitiveLevel).Methods("GET")
	router.HandleFunc("/api/v1/sensitive-levels", api.createSensitiveLevel).Methods("POST")
	router.HandleFunc("/api/v1/sensitive-levels/{id}", api.updateSensitiveLevel).Methods("PUT")
	router.HandleFunc("/api/v1/sensitive-levels/{id}", api.deleteSensitiveLevel).Methods("DELETE")

	// Approval Flow endpoints
	router.HandleFunc("/api/v1/approval-flows", api.listApprovalFlows).Methods("GET")
	router.HandleFunc("/api/v1/approval-flows/{id}", api.getApprovalFlow).Methods("GET")
	router.HandleFunc("/api/v1/approval-flows", api.createApprovalFlow).Methods("POST")
	router.HandleFunc("/api/v1/approval-flows/{id}", api.updateApprovalFlow).Methods("PUT")
	router.HandleFunc("/api/v1/approval-flows/{id}", api.deleteApprovalFlow).Methods("DELETE")

	// Approval Request endpoints
	router.HandleFunc("/api/v1/approval-requests", api.listApprovalRequests).Methods("GET")
	router.HandleFunc("/api/v1/approval-requests/{id}", api.getApprovalRequest).Methods("GET")
	router.HandleFunc("/api/v1/approval-requests", api.createApprovalRequest).Methods("POST")
	router.HandleFunc("/api/v1/approval-requests/{id}/approve", api.approveApprovalRequest).Methods("POST")
	router.HandleFunc("/api/v1/approval-requests/{id}/reject", api.rejectApprovalRequest).Methods("POST")

	// Sensitive Data endpoints
	router.HandleFunc("/api/v1/sensitive-data/detect", api.detectSensitiveData).Methods("POST")
}

// listSensitiveLevels lists all sensitive levels.
func (api *SensitiveApprovalAPI) listSensitiveLevels(w http.ResponseWriter, r *http.Request) {
	req := &v1.ListSensitiveLevelsRequest{}
	resp, err := api.sensitiveApprovalService.ListSensitiveLevels(r.Context(), req)
	if err != nil {
		api.handleError(w, err)
		return
	}

	api.sendResponse(w, resp, http.StatusOK)
}

// getSensitiveLevel gets a sensitive level by ID.
func (api *SensitiveApprovalAPI) getSensitiveLevel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	req := &v1.GetSensitiveLevelRequest{
		Name: fmt.Sprintf("sensitive-levels/%s", id),
	}
	resp, err := api.sensitiveApprovalService.GetSensitiveLevel(r.Context(), req)
	if err != nil {
		api.handleError(w, err)
		return
	}

	api.sendResponse(w, resp, http.StatusOK)
}

// createSensitiveLevel creates a new sensitive level.
func (api *SensitiveApprovalAPI) createSensitiveLevel(w http.ResponseWriter, r *http.Request) {
	var reqBody v1.CreateSensitiveLevelRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.sendError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	resp, err := api.sensitiveApprovalService.CreateSensitiveLevel(r.Context(), &reqBody)
	if err != nil {
		api.handleError(w, err)
		return
	}

	api.sendResponse(w, resp, http.StatusCreated)
}

// updateSensitiveLevel updates an existing sensitive level.
func (api *SensitiveApprovalAPI) updateSensitiveLevel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var reqBody v1.UpdateSensitiveLevelRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.sendError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Set the name in the request body
	reqBody.SensitiveLevel.Name = fmt.Sprintf("sensitive-levels/%s", id)

	// Create field mask
	fieldMask := fieldmaskpb.New()
	if reqBody.SensitiveLevel.DisplayName != "" {
		fieldMask.Paths = append(fieldMask.Paths, "display_name")
	}
	if reqBody.SensitiveLevel.Severity != v1.SensitiveLevel_SEVERITY_UNSPECIFIED {
		fieldMask.Paths = append(fieldMask.Paths, "severity")
	}
	if reqBody.SensitiveLevel.Description != "" {
		fieldMask.Paths = append(fieldMask.Paths, "description")
	}
	if reqBody.SensitiveLevel.Color != "" {
		fieldMask.Paths = append(fieldMask.Paths, "color")
	}
	if len(reqBody.SensitiveLevel.FieldMatchRules) > 0 {
		fieldMask.Paths = append(fieldMask.Paths, "field_match_rules")
	}
	reqBody.UpdateMask = fieldMask

	resp, err := api.sensitiveApprovalService.UpdateSensitiveLevel(r.Context(), &reqBody)
	if err != nil {
		api.handleError(w, err)
		return
	}

	api.sendResponse(w, resp, http.StatusOK)
}

// deleteSensitiveLevel deletes a sensitive level.
func (api *SensitiveApprovalAPI) deleteSensitiveLevel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	req := &v1.DeleteSensitiveLevelRequest{
		Name: fmt.Sprintf("sensitive-levels/%s", id),
	}
	resp, err := api.sensitiveApprovalService.DeleteSensitiveLevel(r.Context(), req)
	if err != nil {
		api.handleError(w, err)
		return
	}

	api.sendResponse(w, resp, http.StatusOK)
}

// listApprovalFlows lists all approval flows.
func (api *SensitiveApprovalAPI) listApprovalFlows(w http.ResponseWriter, r *http.Request) {
	req := &v1.ListApprovalFlowsRequest{}
	resp, err := api.sensitiveApprovalService.ListApprovalFlows(r.Context(), req)
	if err != nil {
		api.handleError(w, err)
		return
	}

	api.sendResponse(w, resp, http.StatusOK)
}

// getApprovalFlow gets an approval flow by ID.
func (api *SensitiveApprovalAPI) getApprovalFlow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	req := &v1.GetApprovalFlowRequest{
		Name: fmt.Sprintf("approval-flows/%s", id),
	}
	resp, err := api.sensitiveApprovalService.GetApprovalFlow(r.Context(), req)
	if err != nil {
		api.handleError(w, err)
		return
	}

	api.sendResponse(w, resp, http.StatusOK)
}

// createApprovalFlow creates a new approval flow.
func (api *SensitiveApprovalAPI) createApprovalFlow(w http.ResponseWriter, r *http.Request) {
	var reqBody v1.CreateApprovalFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.sendError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	resp, err := api.sensitiveApprovalService.CreateApprovalFlow(r.Context(), &reqBody)
	if err != nil {
		api.handleError(w, err)
		return
	}

	api.sendResponse(w, resp, http.StatusCreated)
}

// updateApprovalFlow updates an existing approval flow.
func (api *SensitiveApprovalAPI) updateApprovalFlow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var reqBody v1.UpdateApprovalFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.sendError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Set the name in the request body
	reqBody.ApprovalFlow.Name = fmt.Sprintf("approval-flows/%s", id)

	// Create field mask
	fieldMask := fieldmaskpb.New()
	if reqBody.ApprovalFlow.DisplayName != "" {
		fieldMask.Paths = append(fieldMask.Paths, "display_name")
	}
	if reqBody.ApprovalFlow.Description != "" {
		fieldMask.Paths = append(fieldMask.Paths, "description")
	}
	if reqBody.ApprovalFlow.SensitiveSeverity != v1.SensitiveLevel_SEVERITY_UNSPECIFIED {
		fieldMask.Paths = append(fieldMask.Paths, "sensitive_severity")
	}
	if len(reqBody.ApprovalFlow.Steps) > 0 {
		fieldMask.Paths = append(fieldMask.Paths, "steps")
	}
	reqBody.UpdateMask = fieldMask

	resp, err := api.sensitiveApprovalService.UpdateApprovalFlow(r.Context(), &reqBody)
	if err != nil {
		api.handleError(w, err)
		return
	}

	api.sendResponse(w, resp, http.StatusOK)
}

// deleteApprovalFlow deletes an approval flow.
func (api *SensitiveApprovalAPI) deleteApprovalFlow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	req := &v1.DeleteApprovalFlowRequest{
		Name: fmt.Sprintf("approval-flows/%s", id),
	}
	resp, err := api.sensitiveApprovalService.DeleteApprovalFlow(r.Context(), req)
	if err != nil {
		api.handleError(w, err)
		return
	}

	api.sendResponse(w, resp, http.StatusOK)
}

// listApprovalRequests lists all approval requests.
func (api *SensitiveApprovalAPI) listApprovalRequests(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement list approval requests
	w.WriteHeader(http.StatusNotImplemented)
}

// getApprovalRequest gets an approval request by ID.
func (api *SensitiveApprovalAPI) getApprovalRequest(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get approval request
	w.WriteHeader(http.StatusNotImplemented)
}

// createApprovalRequest creates a new approval request.
func (api *SensitiveApprovalAPI) createApprovalRequest(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement create approval request
	w.WriteHeader(http.StatusNotImplemented)
}

// approveApprovalRequest approves an approval request.
func (api *SensitiveApprovalAPI) approveApprovalRequest(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement approve approval request
	w.WriteHeader(http.StatusNotImplemented)
}

// rejectApprovalRequest rejects an approval request.
func (api *SensitiveApprovalAPI) rejectApprovalRequest(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement reject approval request
	w.WriteHeader(http.StatusNotImplemented)
}

// detectSensitiveData detects sensitive data in a SQL statement.
func (api *SensitiveApprovalAPI) detectSensitiveData(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		SQL            string `json:"sql"`
		DatabaseInstance string `json:"database_instance"`
		Database       string `json:"database"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.sendError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if reqBody.SQL == "" {
		api.sendError(w, http.StatusBadRequest, "SQL is required")
		return
	}

	matches, err := api.sensitiveDataService.DetectSensitiveData(r.Context(), reqBody.SQL, reqBody.DatabaseInstance, reqBody.Database)
	if err != nil {
		api.handleError(w, err)
		return
	}

	api.sendResponse(w, matches, http.StatusOK)
}

// handleError handles gRPC errors.
func (api *SensitiveApprovalAPI) handleError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		api.sendError(w, http.StatusInternalServerError, fmt.Sprintf("internal server error: %v", err))
		return
	}

	switch st.Code() {
	case codes.InvalidArgument:
		api.sendError(w, http.StatusBadRequest, st.Message())
	case codes.NotFound:
		api.sendError(w, http.StatusNotFound, st.Message())
	case codes.PermissionDenied:
		api.sendError(w, http.StatusForbidden, st.Message())
	default:
		api.sendError(w, http.StatusInternalServerError, st.Message())
	}
}

// sendResponse sends a JSON response.
func (api *SensitiveApprovalAPI) sendResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	switch v := data.(type) {
	case *v1.ListSensitiveLevelsResponse:
		_ = json.NewEncoder(w).Encode(v)
	case *v1.SensitiveLevel:
		_ = json.NewEncoder(w).Encode(v)
	case *v1.Empty:
		// No content
	default:
		_ = json.NewEncoder(w).Encode(data)
	}
}

// sendError sends an error response.
func (api *SensitiveApprovalAPI) sendError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
