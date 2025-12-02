package service

import (
	"context"
	"testing"

	v1 "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSensitiveApprovalService is a mock implementation of v1.SensitiveApprovalServiceClient.
type MockSensitiveApprovalService struct {
	mock.Mock
}

func (m *MockSensitiveApprovalService) ListSensitiveLevels(ctx context.Context, req *v1.ListSensitiveLevelsRequest) (*v1.ListSensitiveLevelsResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*v1.ListSensitiveLevelsResponse), args.Error(1)
}

func (m *MockSensitiveApprovalService) GetSensitiveLevel(ctx context.Context, req *v1.GetSensitiveLevelRequest) (*v1.SensitiveLevel, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*v1.SensitiveLevel), args.Error(1)
}

func (m *MockSensitiveApprovalService) CreateSensitiveLevel(ctx context.Context, req *v1.CreateSensitiveLevelRequest) (*v1.SensitiveLevel, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*v1.SensitiveLevel), args.Error(1)
}

func (m *MockSensitiveApprovalService) UpdateSensitiveLevel(ctx context.Context, req *v1.UpdateSensitiveLevelRequest) (*v1.SensitiveLevel, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*v1.SensitiveLevel), args.Error(1)
}

func (m *MockSensitiveApprovalService) DeleteSensitiveLevel(ctx context.Context, req *v1.DeleteSensitiveLevelRequest) (*v1.Empty, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*v1.Empty), args.Error(1)
}

func (m *MockSensitiveApprovalService) ListApprovalFlows(ctx context.Context, req *v1.ListApprovalFlowsRequest) (*v1.ListApprovalFlowsResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*v1.ListApprovalFlowsResponse), args.Error(1)
}

func (m *MockSensitiveApprovalService) GetApprovalFlow(ctx context.Context, req *v1.GetApprovalFlowRequest) (*v1.ApprovalFlow, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*v1.ApprovalFlow), args.Error(1)
}

func (m *MockSensitiveApprovalService) CreateApprovalFlow(ctx context.Context, req *v1.CreateApprovalFlowRequest) (*v1.ApprovalFlow, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*v1.ApprovalFlow), args.Error(1)
}

func (m *MockSensitiveApprovalService) UpdateApprovalFlow(ctx context.Context, req *v1.UpdateApprovalFlowRequest) (*v1.ApprovalFlow, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*v1.ApprovalFlow), args.Error(1)
}

func (m *MockSensitiveApprovalService) DeleteApprovalFlow(ctx context.Context, req *v1.DeleteApprovalFlowRequest) (*v1.Empty, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*v1.Empty), args.Error(1)
}

func TestGetMatchingSensitiveLevels(t *testing.T) {
	// Setup mock
	mockSensitiveApprovalService := new(MockSensitiveApprovalService)
	defer mockSensitiveApprovalService.AssertExpectations(t)

	// Mock data
	sensitiveLevels := []*v1.SensitiveLevel{
		{
			Name:          "sensitive-levels/high",
			DisplayName:   "High Sensitivity",
			Severity:      v1.SensitiveLevel_SEVERITY_HIGH,
			FieldMatchRules: []*v1.FieldMatchRule{
				{
					FieldNameRegex: ".*password.*",
				},
			},
		},
		{
			Name:          "sensitive-levels/medium",
			DisplayName:   "Medium Sensitivity",
			Severity:      v1.SensitiveLevel_SEVERITY_MEDIUM,
			FieldMatchRules: []*v1.FieldMatchRule{
				{
					FieldNameRegex: ".*email.*",
				},
			},
		},
	}

	mockSensitiveApprovalService.On("ListSensitiveLevels", mock.Anything, mock.Anything).Return(&v1.ListSensitiveLevelsResponse{
		SensitiveLevels: sensitiveLevels,
	}, nil)

	// Create service
	service := NewSensitiveDataService(mockSensitiveApprovalService)

	// Test cases
	ctx := context.Background()

	// Test 1: Password field should match high sensitivity
	levels, err := service.GetMatchingSensitiveLevels(ctx, "password", "varchar")
	assert.NoError(t, err)
	assert.Len(t, levels, 1)
	assert.Equal(t, v1.SensitiveLevel_SEVERITY_HIGH, levels[0].Severity)

	// Test 2: Email field should match medium sensitivity
	levels, err = service.GetMatchingSensitiveLevels(ctx, "email", "varchar")
	assert.NoError(t, err)
	assert.Len(t, levels, 1)
	assert.Equal(t, v1.SensitiveLevel_SEVERITY_MEDIUM, levels[0].Severity)

	// Test 3: Name field should not match any
	levels, err = service.GetMatchingSensitiveLevels(ctx, "name", "varchar")
	assert.NoError(t, err)
	assert.Len(t, levels, 0)
}

func TestNeedsApproval(t *testing.T) {
	// Test cases
	assert.True(t, NeedsApproval(v1.SensitiveLevel_SEVERITY_HIGH))
	assert.True(t, NeedsApproval(v1.SensitiveLevel_SEVERITY_MEDIUM))
	assert.False(t, NeedsApproval(v1.SensitiveLevel_SEVERITY_LOW))
	assert.False(t, NeedsApproval(v1.SensitiveLevel_SEVERITY_UNSPECIFIED))
}

func TestGetMaxSensitiveSeverity(t *testing.T) {
	// Test cases
	matches := []*SensitiveDataMatch{
		{
			SensitiveLevel: &v1.SensitiveLevel{
				Severity: v1.SensitiveLevel_SEVERITY_MEDIUM,
			},
		},
		{
			SensitiveLevel: &v1.SensitiveLevel{
				Severity: v1.SensitiveLevel_SEVERITY_HIGH,
			},
		},
	}
	maxSeverity := GetMaxSensitiveSeverity(matches)
	assert.Equal(t, v1.SensitiveLevel_SEVERITY_HIGH, maxSeverity)

	// Test with empty matches
	matches = []*SensitiveDataMatch{}
	maxSeverity = GetMaxSensitiveSeverity(matches)
	assert.Equal(t, v1.SensitiveLevel_SEVERITY_UNSPECIFIED, maxSeverity)
}

func TestMatchesRule(t *testing.T) {
	// Setup mock
	mockSensitiveApprovalService := new(MockSensitiveApprovalService)
	defer mockSensitiveApprovalService.AssertExpectations(t)

	// Create service
	service := NewSensitiveDataService(mockSensitiveApprovalService)

	// Test cases

	// Test 1: Exact name match
	rule := &v1.FieldMatchRule{
		FieldNameRegex: "^password$",
	}
	assert.True(t, service.matchesRule(rule, "password", "varchar"))
	assert.False(t, service.matchesRule(rule, "user_password", "varchar"))

	// Test 2: Regex name match
	rule = &v1.FieldMatchRule{
		FieldNameRegex: ".*password.*",
	}
	assert.True(t, service.matchesRule(rule, "password", "varchar"))
	assert.True(t, service.matchesRule(rule, "user_password", "varchar"))
	assert.True(t, service.matchesRule(rule, "password_hash", "varchar"))
	assert.False(t, service.matchesRule(rule, "pass", "varchar"))

	// Test 3: Field type match
	rule = &v1.FieldMatchRule{
		FieldType: "varchar",
	}
	assert.True(t, service.matchesRule(rule, "any_field", "varchar"))
	assert.False(t, service.matchesRule(rule, "any_field", "int"))

	// Test 4: Combined name and type match
	rule = &v1.FieldMatchRule{
		FieldNameRegex: ".*phone.*",
		FieldType:      "varchar",
	}
	assert.True(t, service.matchesRule(rule, "phone_number", "varchar"))
	assert.False(t, service.matchesRule(rule, "phone_number", "int"))
	assert.False(t, service.matchesRule(rule, "age", "int"))
}
