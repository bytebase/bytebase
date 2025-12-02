// Copyright 2024 Bytebase Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/bytebase/bytebase/backend/api/v1/gen/v1"
    "github.com/bytebase/bytebase/backend/store"
    "github.com/bytebase/bytebase/backend/testutil"
)

func TestCreateSensitiveLevel(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create a test instance
    instance := &store.InstanceMessage{
        ID:          "test-instance",
        ResourceUID: "test-instance-uid",
        Name:        "Test Instance",
        Type:        store.MYSQL,
        Host:        "localhost",
        Port:        3306,
        Database:    "test",
        Status:      store.INSTANCE_STATUS_READY,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    err := s.CreateInstance(ctx, instance)
    require.NoError(t, err)
    
    // Create sensitive level service
    service := NewSensitiveLevelService(s)
    
    // Test create sensitive level
    req := &v1.CreateSensitiveLevelRequest{
        DisplayName: "Test Sensitive Level",
        Description: "Test description",
        Level:       v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        TableName:   "users",
        SchemaName:  "public",
        InstanceId:  "test-instance",
        FieldRules: []*v1.FieldRule{
            {
                FieldName: "password",
                DataType:  "varchar",
                RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_EXACT,
                Pattern:   "password",
            },
        },
    }
    
    resp, err := service.CreateSensitiveLevel(ctx, req)
    require.NoError(t, err)
    require.NotNil(t, resp)
    
    assert.Equal(t, "Test Sensitive Level", resp.SensitiveLevel.DisplayName)
    assert.Equal(t, v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH, resp.SensitiveLevel.Level)
    assert.Equal(t, "users", resp.SensitiveLevel.TableName)
    assert.Equal(t, "public", resp.SensitiveLevel.SchemaName)
    assert.Equal(t, "test-instance", resp.SensitiveLevel.InstanceId)
    assert.Len(t, resp.SensitiveLevel.FieldRules, 1)
    assert.Equal(t, "password", resp.SensitiveLevel.FieldRules[0].FieldName)
}

func TestGetSensitiveLevel(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create a test instance
    instance := &store.InstanceMessage{
        ID:          "test-instance",
        ResourceUID: "test-instance-uid",
        Name:        "Test Instance",
        Type:        store.MYSQL,
        Host:        "localhost",
        Port:        3306,
        Database:    "test",
        Status:      store.INSTANCE_STATUS_READY,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    err := s.CreateInstance(ctx, instance)
    require.NoError(t, err)
    
    // Create sensitive level service
    service := NewSensitiveLevelService(s)
    
    // Create a sensitive level first
    createReq := &v1.CreateSensitiveLevelRequest{
        DisplayName: "Test Sensitive Level",
        Description: "Test description",
        Level:       v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM,
        TableName:   "users",
        SchemaName:  "public",
        InstanceId:  "test-instance",
        FieldRules: []*v1.FieldRule{
            {
                FieldName: "email",
                DataType:  "varchar",
                RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_REGEX,
                Pattern:   ".*@.*",
            },
        },
    }
    
    createResp, err := service.CreateSensitiveLevel(ctx, createReq)
    require.NoError(t, err)
    
    // Test get sensitive level
    getReq := &v1.GetSensitiveLevelRequest{
        SensitiveLevelId: createResp.SensitiveLevel.Name,
    }
    
    getResp, err := service.GetSensitiveLevel(ctx, getReq)
    require.NoError(t, err)
    require.NotNil(t, getResp)
    
    assert.Equal(t, "Test Sensitive Level", getResp.SensitiveLevel.DisplayName)
    assert.Equal(t, v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM, getResp.SensitiveLevel.Level)
    assert.Equal(t, "users", getResp.SensitiveLevel.TableName)
    assert.Equal(t, "public", getResp.SensitiveLevel.SchemaName)
    assert.Equal(t, "test-instance", getResp.SensitiveLevel.InstanceId)
    assert.Len(t, getResp.SensitiveLevel.FieldRules, 1)
    assert.Equal(t, "email", getResp.SensitiveLevel.FieldRules[0].FieldName)
}

func TestListSensitiveLevels(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create a test instance
    instance := &store.InstanceMessage{
        ID:          "test-instance",
        ResourceUID: "test-instance-uid",
        Name:        "Test Instance",
        Type:        store.MYSQL,
        Host:        "localhost",
        Port:        3306,
        Database:    "test",
        Status:      store.INSTANCE_STATUS_READY,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    err := s.CreateInstance(ctx, instance)
    require.NoError(t, err)
    
    // Create sensitive level service
    service := NewSensitiveLevelService(s)
    
    // Create multiple sensitive levels
    for i := 0; i < 3; i++ {
        req := &v1.CreateSensitiveLevelRequest{
            DisplayName: "Test Sensitive Level " + string(rune(i+1)),
            Description: "Test description " + string(rune(i+1)),
            Level:       v1.SensitivityLevel_SENSITIVITY_LEVEL_LOW,
            TableName:   "table" + string(rune(i+1)),
            SchemaName:  "public",
            InstanceId:  "test-instance",
            FieldRules: []*v1.FieldRule{
                {
                    FieldName: "field" + string(rune(i+1)),
                    DataType:  "varchar",
                    RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_EXACT,
                    Pattern:   "field" + string(rune(i+1)),
                },
            },
        }
        
        _, err := service.CreateSensitiveLevel(ctx, req)
        require.NoError(t, err)
    }
    
    // Test list sensitive levels
    listReq := &v1.ListSensitiveLevelsRequest{
        Parent: "instances/test-instance",
    }
    
    listResp, err := service.ListSensitiveLevels(ctx, listReq)
    require.NoError(t, err)
    require.NotNil(t, listResp)
    
    assert.Len(t, listResp.SensitiveLevels, 3)
    for i, level := range listResp.SensitiveLevels {
        assert.Contains(t, level.DisplayName, "Test Sensitive Level")
        assert.Equal(t, v1.SensitivityLevel_SENSITIVITY_LEVEL_LOW, level.Level)
        assert.Contains(t, level.TableName, "table")
    }
}

func TestUpdateSensitiveLevel(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create a test instance
    instance := &store.InstanceMessage{
        ID:          "test-instance",
        ResourceUID: "test-instance-uid",
        Name:        "Test Instance",
        Type:        store.MYSQL,
        Host:        "localhost",
        Port:        3306,
        Database:    "test",
        Status:      store.INSTANCE_STATUS_READY,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    err := s.CreateInstance(ctx, instance)
    require.NoError(t, err)
    
    // Create sensitive level service
    service := NewSensitiveLevelService(s)
    
    // Create a sensitive level first
    createReq := &v1.CreateSensitiveLevelRequest{
        DisplayName: "Test Sensitive Level",
        Description: "Test description",
        Level:       v1.SensitivityLevel_SENSITIVITY_LEVEL_HIGH,
        TableName:   "users",
        SchemaName:  "public",
        InstanceId:  "test-instance",
        FieldRules: []*v1.FieldRule{
            {
                FieldName: "password",
                DataType:  "varchar",
                RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_EXACT,
                Pattern:   "password",
            },
        },
    }
    
    createResp, err := service.CreateSensitiveLevel(ctx, createReq)
    require.NoError(t, err)
    
    // Test update sensitive level
    updateReq := &v1.UpdateSensitiveLevelRequest{
        SensitiveLevelId: createResp.SensitiveLevel.Name,
        SensitiveLevel: &v1.SensitiveLevel{
            DisplayName: "Updated Sensitive Level",
            Description: "Updated description",
            Level:       v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM,
            TableName:   "users",
            SchemaName:  "public",
            InstanceId:  "test-instance",
            FieldRules: []*v1.FieldRule{
                {
                    FieldName: "password",
                    DataType:  "varchar",
                    RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_EXACT,
                    Pattern:   "password",
                },
                {
                    FieldName: "email",
                    DataType:  "varchar",
                    RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_REGEX,
                    Pattern:   ".*@.*",
                },
            },
        },
        UpdateMask: []string{"display_name", "description", "level", "field_rules"},
    }
    
    updateResp, err := service.UpdateSensitiveLevel(ctx, updateReq)
    require.NoError(t, err)
    require.NotNil(t, updateResp)
    
    assert.Equal(t, "Updated Sensitive Level", updateResp.SensitiveLevel.DisplayName)
    assert.Equal(t, "Updated description", updateResp.SensitiveLevel.Description)
    assert.Equal(t, v1.SensitivityLevel_SENSITIVITY_LEVEL_MEDIUM, updateResp.SensitiveLevel.Level)
    assert.Len(t, updateResp.SensitiveLevel.FieldRules, 2)
}

func TestDeleteSensitiveLevel(t *testing.T) {
    ctx := context.Background()
    s := testutil.NewStore(t)
    
    // Create a test instance
    instance := &store.InstanceMessage{
        ID:          "test-instance",
        ResourceUID: "test-instance-uid",
        Name:        "Test Instance",
        Type:        store.MYSQL,
        Host:        "localhost",
        Port:        3306,
        Database:    "test",
        Status:      store.INSTANCE_STATUS_READY,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    err := s.CreateInstance(ctx, instance)
    require.NoError(t, err)
    
    // Create sensitive level service
    service := NewSensitiveLevelService(s)
    
    // Create a sensitive level first
    createReq := &v1.CreateSensitiveLevelRequest{
        DisplayName: "Test Sensitive Level",
        Description: "Test description",
        Level:       v1.SensitivityLevel_SENSITIVITY_LEVEL_LOW,
        TableName:   "users",
        SchemaName:  "public",
        InstanceId:  "test-instance",
        FieldRules: []*v1.FieldRule{
            {
                FieldName: "password",
                DataType:  "varchar",
                RuleType:  v1.MatchingRuleType_MATCHING_RULE_TYPE_EXACT,
                Pattern:   "password",
            },
        },
    }
    
    createResp, err := service.CreateSensitiveLevel(ctx, createReq)
    require.NoError(t, err)
    
    // Test delete sensitive level
    deleteReq := &v1.DeleteSensitiveLevelRequest{
        SensitiveLevelId: createResp.SensitiveLevel.Name,
    }
    
    _, err = service.DeleteSensitiveLevel(ctx, deleteReq)
    require.NoError(t, err)
    
    // Verify deletion
    getReq := &v1.GetSensitiveLevelRequest{
        SensitiveLevelId: createResp.SensitiveLevel.Name,
    }
    
    _, err = service.GetSensitiveLevel(ctx, getReq)
    require.Error(t, err)
}