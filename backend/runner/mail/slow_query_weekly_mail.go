// Package mail contains the slow query weekly mail sender.
package mail

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/mail"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	//go:embed templates/for-dba/need_configure.html
	//go:embed templates/for-dba/environment_header.html
	//go:embed templates/for-dba/environment_footer.html
	//go:embed templates/for-dba/environment_no_instance_configured.html
	//go:embed templates/for-dba/environment_no_slow_query.html
	//go:embed templates/for-dba/header.html
	//go:embed templates/for-dba/footer.html
	//go:embed templates/for-dba/database_table_header.html
	//go:embed templates/for-dba/database_table_item.html
	//go:embed templates/for-dba/database_table_footer.html
	//go:embed templates/for-dba/instance_header.html
	//go:embed templates/for-dba/instance_footer.html
	//go:embed templates/for-project-owner/header.html
	//go:embed templates/for-project-owner/footer.html
	//go:embed templates/for-project-owner/environment_header.html
	//go:embed templates/for-project-owner/environment_footer.html
	//go:embed templates/for-project-owner/table_item.html
	emailTemplates embed.FS
)

// NewSender creates a new slow query weekly mail sender.
func NewSender(store *store.Store, stateCfg *state.State) *SlowQueryWeeklyMailSender {
	return &SlowQueryWeeklyMailSender{
		store:    store,
		stateCfg: stateCfg,
	}
}

// SlowQueryWeeklyMailSender is the slow query weekly mail sender.
type SlowQueryWeeklyMailSender struct {
	store    *store.Store
	stateCfg *state.State
}

// Run will run the slow query weekly mail sender.
func (s *SlowQueryWeeklyMailSender) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug("Slow query weekly mail sender started")
	for {
		select {
		case <-ctx.Done():
			log.Debug("Slow query weekly mail sender received context cancellation")
			return
		case <-ticker.C:
			log.Debug("Slow query weekly mail sender received tick")
			now := time.Now()
			// Send email every Saturday in 00:00 ~ 00:59.
			if now.Weekday() == time.Saturday && now.Hour() == 0 {
				s.sendEmail(ctx, now)
			}
		}
	}
}

func (s *SlowQueryWeeklyMailSender) sendEmail(ctx context.Context, now time.Time) {
	name := api.SettingWorkspaceMailDelivery
	mailSetting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{Name: &name})
	if err != nil {
		log.Error("Failed to get mail setting", zap.Error(err))
		return
	}

	if mailSetting == nil {
		return
	}

	var storeValue storepb.SMTPMailDeliverySetting
	if err := protojson.Unmarshal([]byte(mailSetting.Value), &storeValue); err != nil {
		log.Error("Failed to unmarshal setting value", zap.Error(err))
		return
	}
	apiValue := convertStorepbToAPIMailDeliveryValue(&storeValue)

	consoleRedirectURL := "www.bytebase.com"
	workspaceProfileSettingName := api.SettingWorkspaceProfile
	setting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{Name: &workspaceProfileSettingName})
	if err != nil {
		log.Error("Failed to get workspace profile setting", zap.Error(err))
		return
	}
	if setting != nil {
		settingValue := new(storepb.WorkspaceProfileSetting)
		if err := protojson.Unmarshal([]byte(setting.Value), settingValue); err != nil {
			log.Error("Failed to unmarshal setting value", zap.Error(err))
			return
		}
		if settingValue.ExternalUrl != "" {
			consoleRedirectURL = settingValue.ExternalUrl
		}
	}

	slowQueryPolicyType := api.PolicyTypeSlowQuery
	instanceResourceType := api.PolicyResourceTypeInstance
	policies, err := s.store.ListPoliciesV2(ctx, &store.FindPolicyMessage{
		Type:         &slowQueryPolicyType,
		ResourceType: &instanceResourceType,
	})
	if err != nil {
		log.Error("Failed to list slow query policies", zap.Error(err))
		return
	}

	var activePolicies []*store.PolicyMessage
	for _, policy := range policies {
		payload, err := api.UnmarshalSlowQueryPolicy(policy.Payload)
		if err != nil {
			log.Error("Failed to unmarshal slow query policy payload", zap.Error(err))
			return
		}
		if payload.Active {
			activePolicies = append(activePolicies, policy)
		}
	}

	if len(activePolicies) == 0 {
		dbaRole := api.DBA
		users, err := s.store.ListUsers(ctx, &store.FindUserMessage{Role: &dbaRole})
		if err != nil {
			log.Error("Failed to list dba users", zap.Error(err))
		} else {
			for _, user := range users {
				apiValue.SMTPTo = user.Email
				if err := s.sendNeedConfigSlowQueryPolicyEmail(apiValue, consoleRedirectURL); err != nil {
					log.Error("Failed to send need config slow query policy email", zap.String("user", user.Name), zap.String("email", user.Email), zap.Error(err))
				}
			}
		}

		ownerRole := api.Owner
		users, err = s.store.ListUsers(ctx, &store.FindUserMessage{Role: &ownerRole})
		if err != nil {
			log.Error("Failed to list owner users", zap.Error(err))
		} else {
			for _, user := range users {
				apiValue.SMTPTo = user.Email
				if err := s.sendNeedConfigSlowQueryPolicyEmail(apiValue, consoleRedirectURL); err != nil {
					log.Error("Failed to send need config slow query policy email", zap.String("user", user.Name), zap.String("email", user.Email), zap.Error(err))
				}
			}
		}
		return
	}

	if body, err := s.generateWeeklyEmailForDBA(ctx, activePolicies, now, consoleRedirectURL); err == nil {
		dbaRole := api.DBA
		users, err := s.store.ListUsers(ctx, &store.FindUserMessage{Role: &dbaRole})
		if err != nil {
			log.Error("Failed to list dba users", zap.Error(err))
		} else {
			for _, user := range users {
				apiValue.SMTPTo = user.Email
				if err := send(apiValue, fmt.Sprintf("Database slow query weekly report %s", generateDateRange(now)), body); err != nil {
					log.Error("Failed to send need config slow query policy email", zap.String("user", user.Name), zap.String("email", user.Email), zap.Error(err))
				}
			}
		}
	} else {
		log.Error("Failed to generate weekly email for dba", zap.Error(err))
	}

	projects, err := s.store.ListProjectV2(ctx, &store.FindProjectMessage{})
	if err != nil {
		log.Error("Failed to list projects", zap.Error(err))
		return
	}
	for _, project := range projects {
		body, err := s.generateWeeklyEmailForProject(ctx, project, activePolicies, now, consoleRedirectURL)
		if err != nil {
			log.Error("Failed to generate weekly email for project", zap.Error(err))
			continue
		}
		if len(body) == 0 {
			log.Debug("No slow query found for project", zap.String("project", project.Title))
			continue
		}

		projectPolicy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
		if err != nil {
			log.Error("Failed to get project policy", zap.Error(err))
			continue
		}

		for _, binding := range projectPolicy.Bindings {
			if binding.Role == api.Owner {
				for _, member := range binding.Members {
					apiValue.SMTPTo = member.Email
					subject := fmt.Sprintf("%s database slow query weekly report %s", project.Title, generateDateRange(now))
					if err := send(apiValue, subject, body); err != nil {
						log.Error("Failed to send need config slow query policy email", zap.String("user", member.Name), zap.String("email", member.Email), zap.Error(err))
					}
				}
			}
		}
	}
}

func (s *SlowQueryWeeklyMailSender) generateWeeklyEmailForProject(ctx context.Context, project *store.ProjectMessage, policies []*store.PolicyMessage, now time.Time, visitURL string) (body string, err error) {
	header, err := emailTemplates.ReadFile("templates/for-project-owner/header.html")
	if err != nil {
		return "", err
	}
	footer, err := emailTemplates.ReadFile("templates/for-project-owner/footer.html")
	if err != nil {
		return "", err
	}
	environmentHeader, err := emailTemplates.ReadFile("templates/for-project-owner/environment_header.html")
	if err != nil {
		return "", err
	}
	environmentFooter, err := emailTemplates.ReadFile("templates/for-project-owner/environment_footer.html")
	if err != nil {
		return "", err
	}
	tableItem, err := emailTemplates.ReadFile("templates/for-project-owner/table_item.html")
	if err != nil {
		return "", err
	}

	beginDate := now.AddDate(0, 0, -7)
	endDate := now.AddDate(0, 0, -1)
	var buf bytes.Buffer
	headerString := strings.ReplaceAll(string(header), "{{PROJECT_NAME}}", project.Title)
	headerString = strings.ReplaceAll(headerString, "{{BEGIN_DATE}}", beginDate.UTC().Format("2006.01.02"))
	headerString = strings.ReplaceAll(headerString, "{{END_DATE}}", endDate.UTC().Format("2006.01.02"))
	beginUnix := beginDate.Truncate(24 * time.Hour).Unix()
	endUnix := now.Truncate(24 * time.Hour).Add(-1 * time.Second).Unix()
	projectURL := fmt.Sprintf("%s/slow-query?project=%d&fromTime=%d&toTime=%d", strings.TrimSuffix(visitURL, "/"), project.UID, beginUnix, endUnix)
	headerString = strings.ReplaceAll(headerString, "{{PROJECT_LINK}}", projectURL)
	if _, err := buf.WriteString(headerString); err != nil {
		return "", err
	}
	hasSlowQueryInProject := false
	defer func() {
		if err == nil {
			_, err = buf.Write(footer)
		}
		if !hasSlowQueryInProject {
			body = ""
		}
	}()

	instanceMap := make(map[string]*store.InstanceMessage)
	for _, policy := range policies {
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &policy.ResourceUID})
		if err != nil {
			log.Warn("Failed to get instance", zap.Error(err))
			continue
		}
		instanceMap[instance.ResourceID] = instance
	}

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return "", err
	}

	databaseMap := make(map[string][]*store.DatabaseMessage)
	for _, database := range databases {
		if _, exists := instanceMap[database.InstanceID]; !exists {
			continue
		}
		if list, exists := databaseMap[database.EffectiveEnvironmentID]; exists {
			databaseMap[database.EffectiveEnvironmentID] = append(list, database)
		} else {
			databaseMap[database.EffectiveEnvironmentID] = []*store.DatabaseMessage{database}
		}
	}

	environments, err := s.store.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{})
	if err != nil {
		return "", err
	}
	// Sort environments by order DESC.
	sort.Slice(environments, func(i, j int) bool {
		return environments[i].Order > environments[j].Order
	})

	for _, environment := range environments {
		databases, exists := databaseMap[environment.ResourceID]
		if !exists || len(databases) == 0 {
			continue
		}

		hasSlowQuery := false
		sort.Slice(databases, func(i, j int) bool {
			lEngine := engineOrder(instanceMap[databases[i].InstanceID].Engine)
			rEngine := engineOrder(instanceMap[databases[j].InstanceID].Engine)
			if lEngine != rEngine {
				return lEngine < rEngine
			}
			return databases[i].DatabaseName < databases[j].DatabaseName
		})

		for _, database := range databases {
			instance := instanceMap[database.InstanceID]
			listSlowQuery := &store.ListSlowQueryMessage{
				InstanceUID:  &instance.UID,
				DatabaseUID:  &database.UID,
				StartLogDate: &beginDate,
				EndLogDate:   &endDate,
			}

			logs, err := s.store.ListSlowQuery(ctx, listSlowQuery)
			if err != nil {
				return "", err
			}

			if len(logs) == 0 {
				continue
			}

			if !hasSlowQuery {
				hasSlowQuery = true
				if _, err := buf.WriteString(strings.ReplaceAll(string(environmentHeader), "{{ENVIRONMENT_NAME}}", environment.Title)); err != nil {
					return "", err
				}
			}

			sort.Slice(logs, func(i, j int) bool {
				return logs[i].Statistics.MaximumQueryTime.AsDuration() > logs[j].Statistics.MaximumQueryTime.AsDuration()
			})
			if len(logs) > 5 {
				logs = logs[:5]
			}

			for i, log := range logs {
				var item string
				if i == 0 {
					item = strings.ReplaceAll(string(tableItem), "{{DB_TYPE}}", engineTypeString(instance.Engine))
					item = strings.ReplaceAll(item, "{{DB_NAME}}", database.DatabaseName)
				} else {
					item = strings.ReplaceAll(string(tableItem), "{{DB_TYPE}}", "")
					item = strings.ReplaceAll(item, "{{DB_NAME}}", "")
				}
				item = strings.ReplaceAll(item, "{{SLOW_QUERY}}", log.Statistics.SqlFingerprint)
				if _, err := buf.WriteString(item); err != nil {
					return "", err
				}
			}
		}

		if hasSlowQuery {
			if _, err := buf.Write(environmentFooter); err != nil {
				return "", err
			}
		}

		hasSlowQueryInProject = hasSlowQueryInProject || hasSlowQuery
	}

	return buf.String(), nil
}

func engineTypeString(engine db.Type) string {
	switch engine {
	case db.MySQL:
		return "MySQL"
	case db.Postgres:
		return "Postgres"
	}
	return ""
}

func send(mailSetting *api.SettingWorkspaceMailDeliveryValue, subject string, body string) error {
	email := mail.NewEmailMsg()

	email.SetFrom(fmt.Sprintf("Bytebase <%s>", mailSetting.SMTPFrom)).
		AddTo(mailSetting.SMTPTo).
		SetSubject(subject).
		SetBody(body)
	client := mail.NewSMTPClient(mailSetting.SMTPServerHost, mailSetting.SMTPServerPort)
	client.SetAuthType(convertToMailSMTPAuthType(mailSetting.SMTPAuthenticationType)).
		SetAuthCredentials(mailSetting.SMTPUsername, *mailSetting.SMTPPassword).
		SetEncryptionType(convertToMailSMTPEncryptionType(mailSetting.SMTPEncryptionType))

	if err := client.SendMail(email); err != nil {
		return err
	}
	log.Debug("Successfully sent need configure slow query policy email", zap.String("to", mailSetting.SMTPTo), zap.String("subject", subject))
	return nil
}

func (s *SlowQueryWeeklyMailSender) generateWeeklyEmailForDBA(ctx context.Context, policies []*store.PolicyMessage, now time.Time, visitURL string) (string, error) {
	beginDate := now.AddDate(0, 0, -7)
	endDate := now.AddDate(0, 0, -1)
	var buf bytes.Buffer
	header, err := emailTemplates.ReadFile("templates/for-dba/header.html")
	if err != nil {
		return "", err
	}
	headerString := strings.ReplaceAll(string(header), "{{VISIT_URL}}", visitURL)
	headerString = strings.ReplaceAll(headerString, "{{BEGIN_DATE}}", beginDate.Format("2006.01.02"))
	headerString = strings.ReplaceAll(headerString, "{{END_DATE}}", endDate.Format("2006.01.02"))
	if _, err := buf.WriteString(headerString); err != nil {
		return "", err
	}

	databaseTableHeader, err := emailTemplates.ReadFile("templates/for-dba/database_table_header.html")
	if err != nil {
		return "", err
	}
	databaseTableItem, err := emailTemplates.ReadFile("templates/for-dba/database_table_item.html")
	if err != nil {
		return "", err
	}
	databaseTableFooter, err := emailTemplates.ReadFile("templates/for-dba/database_table_footer.html")
	if err != nil {
		return "", err
	}
	environmentHeader, err := emailTemplates.ReadFile("templates/for-dba/environment_header.html")
	if err != nil {
		return "", err
	}
	environmentFooter, err := emailTemplates.ReadFile("templates/for-dba/environment_footer.html")
	if err != nil {
		return "", err
	}
	environmentNoInstanceConfigured, err := emailTemplates.ReadFile("templates/for-dba/environment_no_instance_configured.html")
	if err != nil {
		return "", err
	}
	environmentNoSlowQuery, err := emailTemplates.ReadFile("templates/for-dba/environment_no_slow_query.html")
	if err != nil {
		return "", err
	}
	instanceHeader, err := emailTemplates.ReadFile("templates/for-dba/instance_header.html")
	if err != nil {
		return "", err
	}
	instanceFooter, err := emailTemplates.ReadFile("templates/for-dba/instance_footer.html")
	if err != nil {
		return "", err
	}

	environments, err := s.store.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{})
	if err != nil {
		return "", err
	}
	// Sort environments by order DESC.
	sort.Slice(environments, func(i, j int) bool {
		return environments[i].Order > environments[j].Order
	})

	instanceMap := make(map[string][]*store.InstanceMessage)

	for _, policy := range policies {
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &policy.ResourceUID})
		if err != nil {
			log.Warn("Failed to get instance", zap.Error(err))
			continue
		}

		if list, exists := instanceMap[instance.EnvironmentID]; exists {
			instanceMap[instance.EnvironmentID] = append(list, instance)
		} else {
			instanceMap[instance.EnvironmentID] = []*store.InstanceMessage{instance}
		}
	}

	for _, environment := range environments {
		instances, exists := instanceMap[environment.ResourceID]
		if !exists {
			instances = nil
		}
		if err := s.generateEnvironmentContent(
			ctx,
			&buf,
			environment,
			instances,
			environmentHeader,
			environmentFooter,
			environmentNoInstanceConfigured,
			environmentNoSlowQuery,
			databaseTableHeader,
			databaseTableItem,
			databaseTableFooter,
			instanceHeader,
			instanceFooter,
			visitURL,
			beginDate,
			now,
		); err != nil {
			return "", err
		}
	}

	footer, err := emailTemplates.ReadFile("templates/for-dba/footer.html")
	if err != nil {
		return "", err
	}
	if _, err := buf.Write(footer); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *SlowQueryWeeklyMailSender) generateEnvironmentContent(
	ctx context.Context,
	buf *bytes.Buffer,
	environment *store.EnvironmentMessage,
	instances []*store.InstanceMessage,
	environmentHeader []byte,
	environmentFooter []byte,
	environmentNoInstanceConfigured []byte,
	environmentNoSlowQuery []byte,
	databaseTableHeader []byte,
	databaseTableItem []byte,
	databaseTableFooter []byte,
	instanceHeader []byte,
	instanceFooter []byte,
	visitURL string,
	beginDate time.Time,
	endDate time.Time,
) (err error) {
	if _, err := buf.WriteString(strings.ReplaceAll(string(environmentHeader), "{{ENV_NAME}}", environment.Title)); err != nil {
		return err
	}
	defer func() {
		if err == nil {
			_, err = buf.Write(environmentFooter)
		}
	}()
	if len(instances) == 0 {
		if _, err := buf.Write(environmentNoInstanceConfigured); err != nil {
			return err
		}
		return nil
	}

	sort.Slice(instances, func(i, j int) bool {
		return engineOrder(instances[i].Engine) < engineOrder(instances[j].Engine)
	})

	hasSlowQueryInEnvironment := false
	for _, instance := range instances {
		databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID})
		if err != nil {
			return err
		}

		sort.Slice(databases, func(i, j int) bool {
			return databases[i].DatabaseName < databases[j].DatabaseName
		})

		hasSlowQueryInInstance := false
		for _, database := range databases {
			listSlowQuery := &store.ListSlowQueryMessage{
				InstanceUID:  &instance.UID,
				DatabaseUID:  &database.UID,
				StartLogDate: &beginDate,
				EndLogDate:   &endDate,
			}

			logs, err := s.store.ListSlowQuery(ctx, listSlowQuery)
			if err != nil {
				return err
			}

			if len(logs) == 0 {
				continue
			}

			if !hasSlowQueryInInstance {
				hasSlowQueryInInstance = true
				beginUnix := beginDate.Truncate(24 * time.Hour).UTC().Unix()
				endUnix := endDate.Truncate(24 * time.Hour).Add(-1 * time.Second).UTC().Unix()
				instanceURL := fmt.Sprintf("%s/slow-query?instance=%d&fromTime=%d&toTime=%d", strings.TrimSuffix(visitURL, "/"), instance.UID, beginUnix, endUnix)
				instanceHeaderString := strings.ReplaceAll(string(instanceHeader), "{{INSTANCE_LINK}}", instanceURL)
				instanceHeaderString = strings.ReplaceAll(instanceHeaderString, "{{INSTANCE_NAME}}", instance.Title)
				if _, err := buf.WriteString(instanceHeaderString); err != nil {
					return err
				}
				if _, err := buf.Write(databaseTableHeader); err != nil {
					return err
				}
			}
			sort.Slice(logs, func(i, j int) bool {
				return logs[i].Statistics.MaximumQueryTime.AsDuration() > logs[j].Statistics.MaximumQueryTime.AsDuration()
			})
			if len(logs) > 5 {
				logs = logs[:5]
			}

			for i, log := range logs {
				var item string
				if i == 0 {
					item = strings.ReplaceAll(string(databaseTableItem), "{{DB_NAME}}", database.DatabaseName)
				} else {
					item = strings.ReplaceAll(string(databaseTableItem), "{{DB_NAME}}", "")
				}
				item = strings.ReplaceAll(item, "{{SLOW_QUERY}}", log.Statistics.SqlFingerprint)
				if _, err := buf.WriteString(item); err != nil {
					return err
				}
			}
		}

		if hasSlowQueryInInstance {
			if _, err := buf.Write(databaseTableFooter); err != nil {
				return err
			}
			if _, err := buf.Write(instanceFooter); err != nil {
				return err
			}
		}

		hasSlowQueryInEnvironment = hasSlowQueryInEnvironment || hasSlowQueryInInstance
	}

	if !hasSlowQueryInEnvironment {
		if _, err := buf.Write(environmentNoSlowQuery); err != nil {
			return err
		}
	}

	return nil
}

func engineOrder(engine db.Type) int {
	switch engine {
	case db.MySQL:
		return 1
	case db.Postgres:
		return 2
	default:
		return 100
	}
}

func (*SlowQueryWeeklyMailSender) sendNeedConfigSlowQueryPolicyEmail(mailSetting *api.SettingWorkspaceMailDeliveryValue, visitURL string) error {
	email := mail.NewEmailMsg()

	needConfigureTemplate, err := emailTemplates.ReadFile("templates/for-dba/need_configure.html")
	if err != nil {
		return err
	}

	body := strings.ReplaceAll(string(needConfigureTemplate), "{{VISIT_URL}}", visitURL)
	body = strings.ReplaceAll(body, "{{DOC_LINK}}", `https://www.bytebase.com/docs/slow-query/overview`)

	email.SetFrom(fmt.Sprintf("Bytebase <%s>", mailSetting.SMTPFrom)).
		AddTo(mailSetting.SMTPTo).
		SetSubject("Configure your database slow query report").
		SetBody(body)
	client := mail.NewSMTPClient(mailSetting.SMTPServerHost, mailSetting.SMTPServerPort)
	client.SetAuthType(convertToMailSMTPAuthType(mailSetting.SMTPAuthenticationType)).
		SetAuthCredentials(mailSetting.SMTPUsername, *mailSetting.SMTPPassword).
		SetEncryptionType(convertToMailSMTPEncryptionType(mailSetting.SMTPEncryptionType))

	if err := client.SendMail(email); err != nil {
		return err
	}
	log.Debug("Successfully sent need configure slow query policy email", zap.String("to", mailSetting.SMTPTo))
	return nil
}

func convertToMailSMTPEncryptionType(encryption storepb.SMTPMailDeliverySetting_Encryption) mail.SMTPEncryptionType {
	switch encryption {
	case storepb.SMTPMailDeliverySetting_ENCRYPTION_NONE:
		return mail.SMTPEncryptionTypeNone
	case storepb.SMTPMailDeliverySetting_ENCRYPTION_SSL_TLS:
		return mail.SMTPEncryptionTypeSSLTLS
	case storepb.SMTPMailDeliverySetting_ENCRYPTION_STARTTLS:
		return mail.SMTPEncryptionTypeSTARTTLS
	}
	return mail.SMTPAuthTypeNone
}

func convertToMailSMTPAuthType(auth storepb.SMTPMailDeliverySetting_Authentication) mail.SMTPAuthType {
	switch auth {
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_NONE:
		return mail.SMTPAuthTypeNone
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_PLAIN:
		return mail.SMTPAuthTypePlain
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_CRAM_MD5:
		return mail.SMTPAuthTypeCRAMMD5
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_LOGIN:
		return mail.SMTPAuthTypeLogin
	}
	return mail.SMTPAuthTypeNone
}

func convertStorepbToAPIMailDeliveryValue(pb *storepb.SMTPMailDeliverySetting) *api.SettingWorkspaceMailDeliveryValue {
	if pb == nil {
		return nil
	}
	password := pb.Password
	value := api.SettingWorkspaceMailDeliveryValue{
		SMTPServerHost:         pb.Server,
		SMTPServerPort:         int(pb.Port),
		SMTPEncryptionType:     pb.Encryption,
		SMTPAuthenticationType: pb.Authentication,
		SMTPUsername:           pb.Username,
		SMTPPassword:           &password,
		SMTPFrom:               pb.From,
	}
	return &value
}

func generateDateRange(now time.Time) string {
	endDate := now.AddDate(0, 0, -1)
	startDate := endDate.AddDate(0, 0, -6)
	return fmt.Sprintf("%s ~ %s", startDate.Format("2006.01.02"), endDate.Format("2006.01.02"))
}
