package services

import (
	"context"
	"errors"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// mockGeometryRepository implements repositories.GeometryRepository for unit testing
type mockGeometryRepository struct {
	mockGeometries []*models.GeometryEntity
	mockErr        error
}

func (m *mockGeometryRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.GeometryEntity, error) {
	if m.mockErr != nil {
		return nil, m.mockErr
	}
	if len(m.mockGeometries) > 0 {
		return m.mockGeometries[0], nil
	}
	return nil, errors.New("not found")
}

func (m *mockGeometryRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.GeometryEntity, error) {
	return m.mockGeometries, m.mockErr
}

func (m *mockGeometryRepository) Search(ctx context.Context, params sqlc.SearchGeometriesParams) ([]*models.GeometryEntity, error) {
	return m.mockGeometries, m.mockErr
}

func (m *mockGeometryRepository) SearchByEntityName(ctx context.Context, params sqlc.SearchGeometriesByEntityNameParams) ([]*models.EntityGeometriesSearchEntity, error) {
	return nil, m.mockErr
}

func (m *mockGeometryRepository) Create(ctx context.Context, params sqlc.CreateGeometryParams) (*models.GeometryEntity, error) {
	return nil, nil
}

func (m *mockGeometryRepository) Update(ctx context.Context, params sqlc.UpdateGeometryParams) (*models.GeometryEntity, error) {
	return nil, nil
}

func (m *mockGeometryRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	return nil
}

func (m *mockGeometryRepository) CreateEntityGeometries(ctx context.Context, params sqlc.CreateEntityGeometriesParams) error {
	return nil
}

func (m *mockGeometryRepository) BulkDeleteEntityGeometriesByEntityId(ctx context.Context, entityId pgtype.UUID) error {
	return nil
}

func (m *mockGeometryRepository) GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.GeometryEntity, error) {
	return m.mockGeometries, m.mockErr
}

func (m *mockGeometryRepository) GetGeometriesByBoundWith(ctx context.Context, boundWith pgtype.UUID) ([]*models.GeometryEntity, error) {
	return m.mockGeometries, m.mockErr
}

func (m *mockGeometryRepository) DeleteByIDs(ctx context.Context, ids []pgtype.UUID) error {
	return nil
}

func (m *mockGeometryRepository) BulkDeleteEntityGeometriesByGeometryID(ctx context.Context, geometryID pgtype.UUID) error {
	return nil
}

func (m *mockGeometryRepository) DeleteEntityGeometry(ctx context.Context, entityID pgtype.UUID, geometryID pgtype.UUID) error {
	return nil
}

func (m *mockGeometryRepository) DeleteEntityGeometriesByProjectID(ctx context.Context, projectID pgtype.UUID) error {
	return nil
}

func (m *mockGeometryRepository) WithTx(tx pgx.Tx) repositories.GeometryRepository {
	return m
}

// Helper to float pointer
func floatPtr(f float64) *float64 {
	return &f
}

// Helper to int pointer
func intPtr(i int32) *int32 {
	return &i
}

func TestGetGeometryByID_InvalidID(t *testing.T) {
	repo := &mockGeometryRepository{}
	svc := NewGeometryService(repo)

	_, fErr := svc.GetGeometryByID(context.Background(), "invalid-uuid-format")
	if fErr == nil {
		t.Fatal("Expected error for invalid UUID format, got nil")
	}
	if fErr.Code != 400 {
		t.Errorf("Expected status 400, got %d", fErr.Code)
	}
}

func TestSearchGeometries_NoBoundingBox(t *testing.T) {
	repo := &mockGeometryRepository{}
	svc := NewGeometryService(repo)

	req := &request.SearchGeometryDto{
		MinLng: nil, // Missing bounding box parameters
	}

	_, fErr := svc.SearchGeometries(context.Background(), req)
	if fErr == nil {
		t.Fatal("Expected error for missing bounding box, got nil")
	}
	if fErr.Code != 400 {
		t.Errorf("Expected status 400, got %d", fErr.Code)
	}
}

func TestSearchGeometries_InvalidBoundingBox(t *testing.T) {
	repo := &mockGeometryRepository{}
	svc := NewGeometryService(repo)

	req := &request.SearchGeometryDto{
		MinLng: floatPtr(105.0),
		MinLat: floatPtr(21.0),
		MaxLng: floatPtr(100.0), // MaxLng < MinLng (Invalid)
		MaxLat: floatPtr(22.0),
	}

	_, fErr := svc.SearchGeometries(context.Background(), req)
	if fErr == nil {
		t.Fatal("Expected error for invalid bounding box coordinates, got nil")
	}
	if fErr.Code != 400 {
		t.Errorf("Expected status 400, got %d", fErr.Code)
	}
	if fErr.Message != "Invalid bounding box coordinates" {
		t.Errorf("Expected error message 'Invalid bounding box coordinates', got '%s'", fErr.Message)
	}
}

func TestSearchGeometries_Valid(t *testing.T) {
	mockGeometries := []*models.GeometryEntity{
		{
			ID:      "87570494-0cfb-4e14-8789-7cfc0245037d",
			GeoType: 3, // Polygon
			Bbox: &response.Bbox{
				MinLng: 102.0, MinLat: 8.0, MaxLng: 109.0, MaxLat: 23.0,
			},
		},
	}
	repo := &mockGeometryRepository{
		mockGeometries: mockGeometries,
	}
	svc := NewGeometryService(repo)

	req := &request.SearchGeometryDto{
		MinLng: floatPtr(100.0),
		MinLat: floatPtr(5.0),
		MaxLng: floatPtr(110.0),
		MaxLat: floatPtr(25.0),
	}

	res, fErr := svc.SearchGeometries(context.Background(), req)
	if fErr != nil {
		t.Fatalf("Expected no error, got %v", fErr)
	}
	if len(res) != 1 {
		t.Fatalf("Expected 1 geometry response, got %d", len(res))
	}
	if res[0].ID != mockGeometries[0].ID {
		t.Errorf("Expected ID %s, got %s", mockGeometries[0].ID, res[0].ID)
	}
}
