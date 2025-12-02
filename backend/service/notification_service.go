package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bytebase/bytebase/common/log"
)

// NotificationService is the service for sending notifications.
type NotificationService interface {
	// Notify sends a notification to a user
	Notify(ctx context.Context, userID string, title string, message string) error
	// NotifyMultiple sends a notification to multiple users
	NotifyMultiple(ctx context.Context, userIDs []string, title string, message string) error
}

// notificationServiceImpl is the implementation of NotificationService.
type notificationServiceImpl struct {
	// TODO: Add dependencies for email, slack, etc.
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService() NotificationService {
	return &notificationServiceImpl{}
}

// Notify sends a notification to a user.
func (s *notificationServiceImpl) Notify(ctx context.Context, userID string, title string, message string) error {
	log.Infof("Sending notification to user %s: %s", userID, title)

	// TODO: Implement actual notification delivery (email, slack, etc.)
	// For now, just log it
	log.Infof("Notification to %s:\nTitle: %s\nMessage: %s", userID, title, message)

	return nil
}

// NotifyMultiple sends a notification to multiple users.
func (s *notificationServiceImpl) NotifyMultiple(ctx context.Context, userIDs []string, title string, message string) error {
	for _, userID := range userIDs {
		err := s.Notify(ctx, userID, title, message)
		if err != nil {
			log.Warnf("Failed to notify user %s: %v", userID, err)
		}
	}

	return nil
}
