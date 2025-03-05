// Package mail contains the slow query weekly mail sender.
package mail

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/mail"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
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
func NewSender(store *store.Store, stateCfg *state.State, iamManager *iam.Manager) *SlowQueryWeeklyMailSender {
	return &SlowQueryWeeklyMailSender{
		store:      store,
		stateCfg:   stateCfg,
		iamManager: iamManager,
	}
}

// SlowQueryWeeklyMailSender is the slow query weekly mail sender.
type SlowQueryWeeklyMailSender struct {
	store      *store.Store
	stateCfg   *state.State
	iamManager *iam.Manager
}

// Run will run the slow query weekly mail sender.
func (s *SlowQueryWeeklyMailSender) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	defer wg.Done()
	slog.Debug("Slow query weekly mail sender started")
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Slow query weekly mail sender received context cancellation")
			return
		case <-ticker.C:
			slog.Debug("Slow query weekly mail sender received tick")
			now := time.Now()
			// Send email every Saturday in 00:00 ~ 00:59.
			if now.Weekday() == time.Saturday && now.Hour() == 0 {
				s.sendEmail(ctx, now)
			}
		}
	}
}

func (s *SlowQueryWeeklyMailSender) sendEmail(ctx context.Context, now time.Time) {
	mailSetting, err := s.store.GetSettingV2(ctx, api.SettingWorkspaceMailDelivery)
	if err != nil {
		slog.Error("Failed to get mail setting", log.BBError(err))
		return
	}

	if mailSetting == nil {
		return
	}

	storeValue := &storepb.SMTPMailDeliverySetting{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(mailSetting.Value), storeValue); err != nil {
		slog.Error("Failed to unmarshal setting value", log.BBError(err))
		return
	}

	consoleRedirectURL := "www.bytebase.com"
	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		slog.Error("Failed to get workspace profile setting", log.BBError(err))
		return
	}
	if setting.ExternalUrl != "" {
		consoleRedirectURL = setting.ExternalUrl
	}

	slowQueryPolicyType := api.PolicyTypeSlowQuery
	instanceResourceType := api.PolicyResourceTypeInstance
	policies, err := s.store.ListPoliciesV2(ctx, &store.FindPolicyMessage{
		Type:         &slowQueryPolicyType,
		ResourceType: &instanceResourceType,
	})
	if err != nil {
		slog.Error("Failed to list slow query policies", log.BBError(err))
		return
	}
	workspaceIAMPolicy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		slog.Error("Failed to get workspace IAM policy", log.BBError(err))
		return
	}

	var activePolicies []*store.PolicyMessage
	for _, policy := range policies {
		payload := &storepb.SlowQueryPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), payload); err != nil {
			slog.Error("Failed to unmarshal slow query policy payload", log.BBError(err))
			return
		}
		if payload.Active {
			activePolicies = append(activePolicies, policy)
		}
	}

	if len(activePolicies) == 0 {
		emailList := s.getEmailListByRoles(ctx, workspaceIAMPolicy.Policy, api.WorkspaceDBA, api.WorkspaceAdmin)
		for _, email := range emailList {
			if err := s.sendNeedConfigSlowQueryPolicyEmail(storeValue, email, consoleRedirectURL); err != nil {
				slog.Error("Failed to send need config slow query policy email", slog.String("email", email), log.BBError(err))
			}
		}
		return
	}

	if body, err := s.generateWeeklyEmailForDBA(ctx, activePolicies, now, consoleRedirectURL); err == nil {
		s.sendEmailToUsersByPolicy(
			ctx,
			api.WorkspaceDBA,
			workspaceIAMPolicy.Policy,
			storeValue,
			fmt.Sprintf("Database slow query weekly report %s", generateDateRange(now)),
			body,
		)
	} else {
		slog.Error("Failed to generate weekly email for dba", log.BBError(err))
	}

	projects, err := s.store.ListProjectV2(ctx, &store.FindProjectMessage{})
	if err != nil {
		slog.Error("Failed to list projects", log.BBError(err))
		return
	}
	for _, project := range projects {
		body, err := s.generateWeeklyEmailForProject(ctx, project, activePolicies, now, consoleRedirectURL)
		if err != nil {
			slog.Error("Failed to generate weekly email for project", log.BBError(err))
			continue
		}
		if len(body) == 0 {
			slog.Debug("No slow query found for project", slog.String("project", project.Title))
			continue
		}

		policyMessage, err := s.store.GetProjectIamPolicy(ctx, project.ResourceID)
		if err != nil {
			slog.Error("Failed to get project policy", log.BBError(err))
			continue
		}

		s.sendEmailToUsersByPolicy(
			ctx,
			api.ProjectOwner,
			policyMessage.Policy,
			storeValue,
			fmt.Sprintf("%s database slow query weekly report %s", project.Title, generateDateRange(now)),
			body,
		)
	}
}

func (s *SlowQueryWeeklyMailSender) getEmailListByRoles(
	ctx context.Context,
	policy *storepb.IamPolicy,
	roles ...api.Role,
) []string {
	usersMap := make(map[string]bool)
	for _, role := range roles {
		users := utils.GetUsersByRoleInIAMPolicy(ctx, s.store, role, policy)
		for _, user := range users {
			if user.Type != api.EndUser {
				continue
			}
			if _, ok := usersMap[user.Email]; ok {
				continue
			}
			usersMap[user.Email] = true
		}
	}
	return slices.Collect(maps.Keys(usersMap))
}

func (s *SlowQueryWeeklyMailSender) sendEmailToUsersByPolicy(
	ctx context.Context,
	role api.Role,
	policy *storepb.IamPolicy,
	deliverySetting *storepb.SMTPMailDeliverySetting,
	subject,
	body string,
) {
	emailList := s.getEmailListByRoles(ctx, policy, role)
	for _, email := range emailList {
		if err := send(deliverySetting, email, subject, body); err != nil {
			slog.Error("Failed to send need config slow query policy email", slog.String("email", email), log.BBError(err))
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
	projectURL := fmt.Sprintf("%s/%s/slow-queries?fromTime=%d&toTime=%d", strings.TrimSuffix(visitURL, "/"), common.FormatProject(project.ResourceID), beginUnix, endUnix)
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
		instanceID, err := common.GetInstanceID(policy.Resource)
		if err != nil {
			return "", err
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			slog.Warn("Failed to get instance", log.BBError(err))
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
			lEngine := engineOrder(instanceMap[databases[i].InstanceID].Metadata.GetEngine())
			rEngine := engineOrder(instanceMap[databases[j].InstanceID].Metadata.GetEngine())
			if lEngine != rEngine {
				return lEngine < rEngine
			}
			return databases[i].DatabaseName < databases[j].DatabaseName
		})

		for _, database := range databases {
			instance := instanceMap[database.InstanceID]
			listSlowQuery := &store.ListSlowQueryMessage{
				InstanceID:   &instance.ResourceID,
				DatabaseName: &database.DatabaseName,
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

			total := totalValue{
				totalCount:     0,
				totalQueryTime: 0,
			}

			for _, log := range logs {
				total.totalCount += log.Statistics.Count
				total.totalQueryTime += log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count)
			}

			if len(logs) > 5 {
				logs = logs[:5]
			}

			for i, log := range logs {
				var item string
				if i == 0 {
					item = strings.ReplaceAll(string(tableItem), "{{DB_TYPE}}", engineTypeString(instance.Metadata.GetEngine()))
					item = strings.ReplaceAll(item, "{{DB_NAME}}", database.DatabaseName)
				} else {
					item = strings.ReplaceAll(string(tableItem), "{{DB_TYPE}}", "")
					item = strings.ReplaceAll(item, "{{DB_NAME}}", "")
				}
				item = strings.ReplaceAll(item, "{{SLOW_QUERY}}", log.Statistics.SqlFingerprint)
				item = strings.ReplaceAll(item, "{{TOTAL_QUERY_COUNT}}", fmt.Sprintf("%d", log.Statistics.Count))
				item = strings.ReplaceAll(item, "{{QUERY_COUNT}}", fmt.Sprintf("%.2f%%", (float64(log.Statistics.Count)/float64(total.totalCount))*100))
				item = strings.ReplaceAll(item, "{{MAX_QUERY_TIME}}", durationText(log.Statistics.MaximumQueryTime))
				item = strings.ReplaceAll(item, "{{AVG_QUERY_TIME}}", durationText(log.Statistics.AverageQueryTime))
				item = strings.ReplaceAll(item, "{{QUERY_TIME}}", fmt.Sprintf("%.2f%%", (float64(log.Statistics.AverageQueryTime.AsDuration()*time.Duration(log.Statistics.Count))/float64(total.totalQueryTime))*100))
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

type totalValue struct {
	totalQueryTime time.Duration
	totalCount     int64
}

func durationText(duration *durationpb.Duration) string {
	if duration == nil {
		return "-"
	}
	secs := duration.Seconds
	nanos := duration.Nanos
	total := float64(secs) + float64(nanos/1e9)
	return fmt.Sprintf("%.2fs", total)
}

func engineTypeString(engine storepb.Engine) string {
	switch engine {
	case storepb.Engine_MYSQL:
		return "MySQL"
	case storepb.Engine_POSTGRES:
		return "Postgres"
	}
	return ""
}

func send(mailSetting *storepb.SMTPMailDeliverySetting, to, subject, body string) error {
	email := mail.NewEmailMsg()

	email.SetFrom(fmt.Sprintf("Bytebase <%s>", mailSetting.From)).
		AddTo(to).
		SetSubject(subject).
		SetBody(body)
	client := mail.NewSMTPClient(mailSetting.Server, int(mailSetting.Port))
	client.SetAuthType(convertToMailSMTPAuthType(mailSetting.Authentication)).
		SetAuthCredentials(mailSetting.Username, mailSetting.Password).
		SetEncryptionType(convertToMailSMTPEncryptionType(mailSetting.Encryption))

	if err := client.SendMail(email); err != nil {
		return err
	}
	slog.Debug("Successfully sent need configure slow query policy email", slog.String("to", to), slog.String("subject", subject))
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
		instanceID, err := common.GetInstanceID(policy.Resource)
		if err != nil {
			return "", err
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			slog.Warn("Failed to get instance", log.BBError(err))
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
		return engineOrder(instances[i].Metadata.GetEngine()) < engineOrder(instances[j].Metadata.GetEngine())
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
				InstanceID:   &instance.ResourceID,
				DatabaseName: &database.DatabaseName,
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
				instanceURL := fmt.Sprintf("%s/slow-query?instance=%s&fromTime=%d&toTime=%d", strings.TrimSuffix(visitURL, "/"), instance.ResourceID, beginUnix, endUnix)
				instanceHeaderString := strings.ReplaceAll(string(instanceHeader), "{{INSTANCE_LINK}}", instanceURL)
				instanceHeaderString = strings.ReplaceAll(instanceHeaderString, "{{INSTANCE_NAME}}", instance.Metadata.GetTitle())
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

			total := totalValue{
				totalCount:     0,
				totalQueryTime: 0,
			}

			for _, log := range logs {
				total.totalCount += log.Statistics.Count
				total.totalQueryTime += log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count)
			}

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
				item = strings.ReplaceAll(item, "{{TOTAL_QUERY_COUNT}}", fmt.Sprintf("%d", log.Statistics.Count))
				item = strings.ReplaceAll(item, "{{QUERY_COUNT}}", fmt.Sprintf("%.2f%%", (float64(log.Statistics.Count)/float64(total.totalCount))*100))
				item = strings.ReplaceAll(item, "{{MAX_QUERY_TIME}}", durationText(log.Statistics.MaximumQueryTime))
				item = strings.ReplaceAll(item, "{{AVG_QUERY_TIME}}", durationText(log.Statistics.AverageQueryTime))
				item = strings.ReplaceAll(item, "{{QUERY_TIME}}", fmt.Sprintf("%.2f%%", (float64(log.Statistics.AverageQueryTime.AsDuration()*time.Duration(log.Statistics.Count))/float64(total.totalQueryTime))*100))
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

func engineOrder(engine storepb.Engine) int {
	switch engine {
	case storepb.Engine_MYSQL:
		return 1
	case storepb.Engine_POSTGRES:
		return 2
	default:
		return 100
	}
}

func (*SlowQueryWeeklyMailSender) sendNeedConfigSlowQueryPolicyEmail(mailSetting *storepb.SMTPMailDeliverySetting, to, visitURL string) error {
	email := mail.NewEmailMsg()

	needConfigureTemplate, err := emailTemplates.ReadFile("templates/for-dba/need_configure.html")
	if err != nil {
		return err
	}

	body := strings.ReplaceAll(string(needConfigureTemplate), "{{VISIT_URL}}", visitURL)
	body = strings.ReplaceAll(body, "{{DOC_LINK}}", `https://www.bytebase.com/docs/slow-query/overview`)

	email.SetFrom(fmt.Sprintf("Bytebase <%s>", mailSetting.From)).
		AddTo(to).
		SetSubject("Configure your database slow query report").
		SetBody(body)
	client := mail.NewSMTPClient(mailSetting.Server, int(mailSetting.Port))
	client.SetAuthType(convertToMailSMTPAuthType(mailSetting.Authentication)).
		SetAuthCredentials(mailSetting.Username, mailSetting.Password).
		SetEncryptionType(convertToMailSMTPEncryptionType(mailSetting.Encryption))

	if err := client.SendMail(email); err != nil {
		return err
	}
	slog.Debug("Successfully sent need configure slow query policy email", slog.String("to", to))
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

func generateDateRange(now time.Time) string {
	endDate := now.AddDate(0, 0, -1)
	startDate := endDate.AddDate(0, 0, -6)
	return fmt.Sprintf("%s ~ %s", startDate.Format("2006.01.02"), endDate.Format("2006.01.02"))
}
