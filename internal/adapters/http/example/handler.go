package example

import (
	"encoding/json"
	"errors"
	httpErrors "microservice/internal/platform/http"
	"microservice/internal/platform/logger"
	"microservice/internal/platform/validator"
	"net/http"

	"github.com/go-chi/chi/v5"

	"microservice/internal/adapters/http/response"
	"microservice/internal/core/domain/example"
)

type Handler struct {
	manager  Manager
	validate validator.Validator
}

func NewHandler(manager Manager, validate validator.Validator) *Handler {
	return &Handler{
		manager:  manager,
		validate: validate,
	}
}

func (h *Handler) mapDomainError(err error) error {
	switch {
	case errors.Is(err, example.ErrEntityNotFound):
		return httpErrors.NewNotFound("Entity not found", err)
	case errors.Is(err, example.ErrInvalidEntityID):
		return httpErrors.NewBadRequest("Invalid entity ID", err)
	case errors.Is(err, example.ErrInvalidEmail):
		return httpErrors.NewBadRequest("Invalid email format", err)
	case errors.Is(err, example.ErrInvalidName):
		return httpErrors.NewBadRequest("Invalid name", err)
	case errors.Is(err, example.ErrReservedName):
		return httpErrors.NewBadRequest("Name is reserved", err)
	default:
		var alreadyExistsErr *example.AlreadyExistsError
		if errors.As(err, &alreadyExistsErr) {
			return httpErrors.NewConflict("Entity already exists", err)
		}
		return err
	}
}

func (h *Handler) GetEntity(w http.ResponseWriter, r *http.Request) error {
	entityID := chi.URLParam(r, "id")

	entity, err := h.manager.GetEntity(r.Context(), entityID)
	if err != nil {
		return h.mapDomainError(err)
	}

	response.RespondJSON(w, http.StatusOK, entity)
	return nil
}

type CreateEntityRequest struct {
	ID    string `json:"id" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name" validate:"required"`
}

func (h *Handler) CreateEntity(w http.ResponseWriter, r *http.Request) error {
	contextLogger := logger.FromContext(r.Context())

	var req CreateEntityRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		contextLogger.Warn("Failed to decode request body", logger.Error(err))
		response.RespondError(w, http.StatusBadRequest, errors.New("invalid request payload"))
		return nil
	}

	if err := h.validate.Validate(req); err != nil {
		var validationErr validator.ValidationError
		if errors.As(err, &validationErr) {
			contextLogger.Warn("Validation failed", logger.Error(err))
			response.RespondJSON(w, http.StatusBadRequest, validationErr)
		} else {
			contextLogger.Error("Unexpected validation error", logger.Error(err))
			response.RespondError(w, http.StatusBadRequest, errors.New("invalid request data"))
		}
		return nil
	}

	entity, err := h.manager.CreateEntity(r.Context(), req.ID, req.Email, req.Name)
	if err != nil {
		return h.mapDomainError(err)
	}

	response.RespondJSON(w, http.StatusCreated, entity)
	return nil
}
