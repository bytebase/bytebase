package sampleprojectinstance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestCloudSQLSampleProjectInstanceSmoke(t *testing.T) {
	if os.Getenv("BB_RUN_CLOUD_SQL_SAMPLE_PROJECT_INSTANCE_SMOKE") != "1" {
		t.Skip("cloud SQL sample Project Instance smoke is disabled")
	}
	targetURL := os.Getenv("SAMPLE_PROJECT_INSTANCE_PG_URL")
	if targetURL == "" {
		t.Skip("cloud SQL sample Project Instance target is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	target, err := NewTarget(targetURL)
	if err != nil {
		t.Fatal("cloud SQL sample Project Instance target is invalid")
	}
	if err := target.Validate(ctx); err != nil {
		t.Fatal("cloud SQL sample Project Instance target is not ready")
	}

	token, err := randomSmokeToken()
	if err != nil {
		t.Fatal("cloud SQL sample Project Instance allocation could not be generated")
	}
	allocation := Allocation{
		Database: "bbsi_" + token,
		Role:     "bbsi_r_" + token,
		Password: token + token,
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Minute)
		defer cleanupCancel()
		_ = target.Remove(cleanupCtx, allocation)
	})

	if err := target.Provision(ctx, allocation); err != nil {
		t.Fatal("cloud SQL sample Project Instance provisioning failed")
	}
	sampleConfig := target.config.Copy()
	sampleConfig.Database = allocation.Database
	sampleConfig.User = allocation.Role
	sampleConfig.Password = allocation.Password
	session, err := pgx.ConnectConfig(ctx, sampleConfig)
	if err != nil {
		t.Fatal("cloud SQL sample Project Instance role cannot connect")
	}
	defer session.Close(context.Background())
	if err := session.Ping(ctx); err != nil {
		t.Fatal("cloud SQL sample Project Instance role is unavailable")
	}

	if err := target.Remove(ctx, allocation); err != nil {
		t.Fatal("cloud SQL sample Project Instance cleanup failed")
	}
	if err := session.Ping(ctx); err == nil {
		t.Fatal("cloud SQL sample Project Instance session remained connected")
	}

	control, err := target.connect(ctx, "", "", "")
	if err != nil {
		t.Fatal("cloud SQL sample Project Instance cleanup could not be verified")
	}
	defer control.Close(ctx)
	databasePresent, err := databaseExists(ctx, control, allocation.Database)
	if err != nil || databasePresent {
		t.Fatal("cloud SQL sample Project Instance database remains after cleanup")
	}
	rolePresent, err := roleExists(ctx, control, allocation.Role)
	if err != nil || rolePresent {
		t.Fatal("cloud SQL sample Project Instance role remains after cleanup")
	}
	if err := target.Remove(ctx, allocation); err != nil {
		t.Fatal("cloud SQL sample Project Instance cleanup is not idempotent")
	}
}

func randomSmokeToken() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
