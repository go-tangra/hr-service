package service

import (
	"context"
	"fmt"
	"html"
	"strings"

	grpcMD "google.golang.org/grpc/metadata"

	notificationv1 "buf.build/gen/go/go-tangra/notification/protocolbuffers/go/notification/service/v1"

	"github.com/go-tangra/go-tangra-hr/internal/data/ent"
)

// ensureRejectionTemplate resolves (or creates) the leave rejection notification template.
func (s *LeaveService) ensureRejectionTemplate(ctx context.Context) (string, error) {
	s.rejectTemplateMu.Lock()
	defer s.rejectTemplateMu.Unlock()

	if s.rejectTemplateDone {
		return s.rejectTemplateID, nil
	}

	s.log.Info("Resolving notification template for leave rejection...")

	platformCtx := detachedPlatformContext(ctx)

	tmpl, err := s.notificationClient.FindTemplateByName(platformCtx, rejectionTemplateName)
	if err != nil {
		return "", fmt.Errorf("search rejection template: %w", err)
	}
	if tmpl != nil {
		s.rejectTemplateID = tmpl.GetId()
		s.rejectTemplateDone = true
		s.log.Infof("Found existing rejection template: %s", s.rejectTemplateID)
		return s.rejectTemplateID, nil
	}

	channelID, err := s.notificationClient.FindChannelByName(platformCtx, notificationChannelName)
	if err != nil {
		return "", fmt.Errorf("find channel %q: %w", notificationChannelName, err)
	}

	createReq := &notificationv1.CreateTemplateRequest{
		Name:      rejectionTemplateName,
		ChannelId: channelID,
		Subject:   defaultRejectionSubject,
		Body:      defaultRejectionBodyTemplate,
		Variables: "RecipientName,AbsenceTypeName,StartDate,EndDate,Days,ReviewNotes,ReviewerName",
		IsDefault: false,
	}
	created, err := s.notificationClient.CreateTemplate(platformCtx, createReq)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			s.log.Info("Rejection template already exists, retrying lookup...")
			tmpl2, findErr := s.notificationClient.FindTemplateByName(platformCtx, rejectionTemplateName)
			if findErr != nil {
				return "", fmt.Errorf("retry find rejection template: %w", findErr)
			}
			if tmpl2 != nil {
				s.rejectTemplateID = tmpl2.GetId()
				s.rejectTemplateDone = true
				s.log.Infof("Found rejection template on retry: %s", s.rejectTemplateID)
				return s.rejectTemplateID, nil
			}
		}
		return "", fmt.Errorf("create rejection template: %w", err)
	}
	s.rejectTemplateID = created.GetId()
	s.rejectTemplateDone = true
	s.log.Infof("Created rejection template: %s", s.rejectTemplateID)
	return s.rejectTemplateID, nil
}

// sendRejectionEmail sends a rejection notification email to the leave requester.
// Designed to run in a goroutine — uses detached context.
func (s *LeaveService) sendRejectionEmail(ctx context.Context, entity *ent.LeaveRequest, reviewerName, reviewNotes string) {
	if entity.UserEmail == "" {
		s.log.Warnf("No email for leave request %s, skipping rejection notification", entity.ID)
		return
	}

	templateID, err := s.ensureRejectionTemplate(ctx)
	if err != nil {
		s.log.Errorf("Failed to ensure rejection template: %v", err)
		return
	}

	absenceTypeName := ""
	if entity.Edges.AbsenceType != nil {
		absenceTypeName = entity.Edges.AbsenceType.Name
	}

	variables := map[string]string{
		"RecipientName":   html.EscapeString(entity.UserName),
		"AbsenceTypeName": html.EscapeString(absenceTypeName),
		"StartDate":       entity.StartDate.Format("2006-01-02"),
		"EndDate":         entity.EndDate.Format("2006-01-02"),
		"Days":            fmt.Sprintf("%.1f", entity.Days),
		"ReviewNotes":     html.EscapeString(reviewNotes),
		"ReviewerName":    html.EscapeString(reviewerName),
	}

	platformCtx := detachedPlatformContext(ctx)
	if _, err := s.notificationClient.SendNotification(platformCtx, templateID, entity.UserEmail, variables); err != nil {
		s.log.Errorf("Failed to send rejection email for leave %s: %v", entity.ID, err)
		return
	}
	s.log.Infof("Rejection email sent to %s for leave request %s", entity.UserEmail, entity.ID)
}

// detachedPlatformContext creates a detached gRPC outgoing context with platform tenant (0)
// for notification service calls. Preserves auth metadata from the original context.
func detachedPlatformContext(ctx context.Context) context.Context {
	outMD := grpcMD.New(map[string]string{
		"x-md-global-tenant-id": "0",
	})

	if inMD, ok := grpcMD.FromIncomingContext(ctx); ok {
		for _, key := range []string{"x-md-global-user-id", "x-md-global-username", "x-md-global-roles"} {
			if vals := inMD.Get(key); len(vals) > 0 {
				outMD.Set(key, vals[0])
			}
		}
	}

	return grpcMD.NewOutgoingContext(context.Background(), outMD)
}
