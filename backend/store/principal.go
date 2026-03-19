package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// FindUserMessage is the message for finding users.
type FindUserMessage struct {
	ID          *int
	Email       *string
	ShowDeleted bool
	Limit       *int
	Offset      *int
	FilterQ     *qb.Query
	ProjectID   *string
	// Workspace is required when ProjectID is set, for the project member CTE query.
	Workspace string
}

// UpdateUserMessage is the message to update a user.
type UpdateUserMessage struct {
	Email        *string
	Name         *string
	PasswordHash *string
	Delete       *bool
	MFAConfig    *storepb.MFAConfig
	Profile      *storepb.UserProfile
	Phone        *string
}

// UserMessage is the message for an end user (principal table).
type UserMessage struct {
	ID int
	// Email must be lower case.
	Email         string
	Name          string
	Type          storepb.PrincipalType
	PasswordHash  string
	MemberDeleted bool
	MFAConfig     *storepb.MFAConfig
	Profile       *storepb.UserProfile
	// Phone conforms E.164 format.
	Phone string
	// output only
	CreatedAt time.Time
}

// GetUserByID gets the user by ID.
func (s *Store) GetUserByID(ctx context.Context, id int) (*UserMessage, error) {
	users, err := s.ListUsers(ctx, &FindUserMessage{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	return users[0], nil
}

// GetUserByEmail gets the user by email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*UserMessage, error) {
	if v, ok := s.userEmailCache.Get(email); ok && s.enableCache {
		return v, nil
	}

	if err := s.listAndCacheAllUsers(ctx); err != nil {
		return nil, err
	}

	user, _ := s.userEmailCache.Get(email)
	return user, nil
}

// BatchGetUsersByEmails gets users (of any type) by emails in batch.
func (s *Store) BatchGetUsersByEmails(ctx context.Context, workspace string, emails []string) ([]*UserMessage, error) {
	if len(emails) == 0 {
		return nil, nil
	}

	normalizedEmails := make([]string, len(emails))
	for i, email := range emails {
		normalizedEmails[i] = strings.ToLower(email)
	}

	var users []*UserMessage

	q := qb.Q().Space(`
			SELECT p.id, p.deleted, p.email, p.name, p.password_hash, p.mfa_config, p.phone, p.profile, p.created_at
			FROM principal p
			JOIN policy pol ON pol.workspace = ? AND pol.resource_type = 'WORKSPACE' AND pol.type = 'IAM'
			WHERE p.email = ANY(?)
			  AND EXISTS (
				SELECT 1
				FROM jsonb_array_elements(pol.payload->'bindings') AS binding,
				     jsonb_array_elements_text(binding->'members') AS member
				WHERE member = 'users/' || p.email OR member = ?
			  )
			ORDER BY p.created_at ASC
		`, workspace, normalizedEmails, common.AllUsers)
	sqlStr, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	rows, err := s.GetDB().QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var user UserMessage
		var mfaConfigBytes, profileBytes []byte
		if err := rows.Scan(&user.ID, &user.MemberDeleted, &user.Email, &user.Name, &user.PasswordHash, &mfaConfigBytes, &user.Phone, &profileBytes, &user.CreatedAt); err != nil {
			return nil, err
		}
		user.Type = storepb.PrincipalType_END_USER
		mfaConfig := storepb.MFAConfig{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(mfaConfigBytes, &mfaConfig); err != nil {
			return nil, err
		}
		user.MFAConfig = &mfaConfig
		profile := storepb.UserProfile{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(profileBytes, &profile); err != nil {
			return nil, err
		}
		user.Profile = &profile
		users = append(users, &user)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to scan rows")
	}

	for _, user := range users {
		s.userEmailCache.Add(user.Email, user)
	}
	return users, nil
}

// ListUsers list users.
func (s *Store) ListUsers(ctx context.Context, find *FindUserMessage) ([]*UserMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	users, err := listUserImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, user := range users {
		s.userEmailCache.Add(user.Email, user)
	}
	return users, nil
}

// listAndCacheAllUsers is used for caching all users.
func (s *Store) listAndCacheAllUsers(ctx context.Context) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	users, err := listUserImpl(ctx, tx, &FindUserMessage{ShowDeleted: true})
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	for _, user := range users {
		s.userEmailCache.Add(user.Email, user)
	}
	return nil
}

func listUserImpl(ctx context.Context, txn *sql.Tx, find *FindUserMessage) ([]*UserMessage, error) {
	with := qb.Q()
	from := qb.Q().Space("principal")
	where := qb.Q().Space("TRUE")

	// Build CTE for project filtering if needed
	if v := find.ProjectID; v != nil {
		with.Space(`WITH all_members AS (
			SELECT
				jsonb_array_elements_text(jsonb_array_elements(policy.payload->'bindings')->'members') AS member,
				jsonb_array_elements(policy.payload->'bindings')->>'role' AS role
			FROM policy
			WHERE ((resource_type = ? AND resource = ?) OR resource_type = ?) AND type = ? AND policy.workspace = ?
		),
		project_members AS (
			SELECT ARRAY_AGG(member) AS members FROM all_members WHERE role NOT LIKE 'roles/workspace%'
		)`, storepb.Policy_PROJECT.String(), "projects/"+*v, storepb.Policy_WORKSPACE.String(), storepb.Policy_IAM.String(), find.Workspace)
		from.Space(`INNER JOIN project_members ON (CONCAT('users/', principal.email) = ANY(project_members.members) OR ? = ANY(project_members.members))`, common.AllUsers)
	}

	if filterQ := find.FilterQ; filterQ != nil {
		where.And("?", filterQ)
	}
	if v := find.ID; v != nil {
		where.And("principal.id = ?", *v)
	}
	if v := find.Email; v != nil {
		if *v == common.AllUsers {
			where.And("principal.email = ?", *v)
		} else {
			where.And("principal.email = ?", strings.ToLower(*v))
		}
	}
	if !find.ShowDeleted {
		where.And("principal.deleted = ?", false)
	}

	q := qb.Q().Space("?", with)
	q.Space(`
		SELECT
			principal.id AS user_id,
			principal.deleted,
			principal.email,
			principal.name,
			principal.password_hash,
			principal.mfa_config,
			principal.phone,
			principal.profile,
			principal.created_at
		FROM ?
		WHERE ?
		ORDER BY created_at ASC
	`, from, where)

	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var userMessages []*UserMessage
	rows, err := txn.QueryContext(ctx, sql, args...) // NOSONAR: query is parameterized via qb.Query
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userMessage UserMessage
		var mfaConfigBytes []byte
		var profileBytes []byte
		if err := rows.Scan(
			&userMessage.ID,
			&userMessage.MemberDeleted,
			&userMessage.Email,
			&userMessage.Name,
			&userMessage.PasswordHash,
			&mfaConfigBytes,
			&userMessage.Phone,
			&profileBytes,
			&userMessage.CreatedAt,
		); err != nil {
			return nil, err
		}
		userMessage.Type = storepb.PrincipalType_END_USER

		mfaConfig := storepb.MFAConfig{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(mfaConfigBytes, &mfaConfig); err != nil {
			return nil, err
		}
		userMessage.MFAConfig = &mfaConfig
		profile := storepb.UserProfile{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(profileBytes, &profile); err != nil {
			return nil, err
		}
		userMessage.Profile = &profile

		userMessages = append(userMessages, &userMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userMessages, nil
}

// scanPrincipalRow scans a principal row into a UserMessage (without groups).
func scanPrincipalRow(ctx context.Context, tx *sql.Tx, sqlStr string, args []any) (*UserMessage, error) {
	var user UserMessage
	var mfaConfigBytes []byte
	var profileBytes []byte
	if err := tx.QueryRowContext(ctx, sqlStr, args...).Scan( // NOSONAR: query is parameterized via qb.Query
		&user.ID,
		&user.MemberDeleted,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&mfaConfigBytes,
		&user.Phone,
		&profileBytes,
		&user.CreatedAt,
	); err != nil {
		return nil, err
	}
	user.Type = storepb.PrincipalType_END_USER

	mfaConfig := storepb.MFAConfig{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(mfaConfigBytes, &mfaConfig); err != nil {
		return nil, err
	}
	user.MFAConfig = &mfaConfig

	profile := storepb.UserProfile{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(profileBytes, &profile); err != nil {
		return nil, err
	}
	user.Profile = &profile

	return &user, nil
}

// CreateUser creates an user.
func (s *Store) CreateUser(ctx context.Context, create *UserMessage) (*UserMessage, error) {
	// Double check the passing-in emails.
	// We use lower-case for emails.
	if create.Email != strings.ToLower(create.Email) {
		return nil, errors.Errorf("emails must be lower-case when they are passed into store")
	}

	if create.Profile == nil {
		create.Profile = &storepb.UserProfile{}
	}
	profileBytes, err := protojson.Marshal(create.Profile)
	if err != nil {
		return nil, err
	}

	q := qb.Q().Space(`
		INSERT INTO principal (
			email,
			name,
			password_hash,
			phone,
			profile
		)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, created_at
	`, create.Email, create.Name, create.PasswordHash, create.Phone, profileBytes)

	sqlStr, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	user := &UserMessage{
		Email:        create.Email,
		Name:         create.Name,
		Type:         storepb.PrincipalType_END_USER,
		PasswordHash: create.PasswordHash,
		Phone:        create.Phone,
		CreatedAt:    create.CreatedAt,
		Profile:      create.Profile,
		MFAConfig:    &storepb.MFAConfig{},
	}

	if err := s.GetDB().QueryRowContext(ctx, sqlStr, args...).Scan(&user.ID, &user.CreatedAt); err != nil {
		return nil, err
	}

	s.userEmailCache.Add(user.Email, user)
	return user, nil
}

// UpdateUser updates a user.
func (s *Store) UpdateUser(ctx context.Context, currentUser *UserMessage, patch *UpdateUserMessage) (*UserMessage, error) {
	set := qb.Q()
	if v := patch.Delete; v != nil {
		set.Comma("deleted = ?", *v)
	}
	if v := patch.Name; v != nil {
		set.Comma("name = ?", *v)
	}
	if v := patch.PasswordHash; v != nil {
		set.Comma("password_hash = ?", *v)
		if patch.Profile == nil {
			patch.Profile = currentUser.Profile
			patch.Profile.LastChangePasswordTime = timestamppb.New(time.Now())
		}
	}
	if v := patch.Phone; v != nil {
		set.Comma("phone = ?", *v)
	}
	if v := patch.MFAConfig; v != nil {
		mfaConfigBytes, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		set.Comma("mfa_config = ?", mfaConfigBytes)
	}
	if v := patch.Profile; v != nil {
		profileBytes, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		set.Comma("profile = ?", profileBytes)
	}

	if set.Len() == 0 {
		return currentUser, nil
	}

	sql, args, err := qb.Q().Space(`UPDATE principal SET ? WHERE id = ?
		RETURNING id, deleted, email, name, password_hash, mfa_config, phone, profile, created_at`,
		set, currentUser.ID).ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	updatedUser, err := scanPrincipalRow(ctx, tx, sql, args)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.userEmailCache.Remove(currentUser.Email)
	s.userEmailCache.Add(updatedUser.Email, updatedUser)
	return updatedUser, nil
}

// UpdateUserEmail updates a user's email and all related references.
func (s *Store) UpdateUserEmail(ctx context.Context, user *UserMessage, newEmail string) (*UserMessage, error) {
	newEmail = strings.ToLower(newEmail)

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Update Principal table
	query := qb.Q().Space(`UPDATE principal SET email = ? WHERE id = ?
		RETURNING id, deleted, email, name, password_hash, mfa_config, phone, profile, created_at`,
		newEmail, user.ID)
	sqlStr, args, err := query.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build update principal sql")
	}

	updatedUser, err := scanPrincipalRow(ctx, tx, sqlStr, args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update principal email")
	}

	// 1b. Update creator/deleter columns that previously relied on ON UPDATE CASCADE.
	creatorUpdates := []string{
		"UPDATE plan SET creator = $1 WHERE creator = $2",
		"UPDATE task_run SET creator = $1 WHERE creator = $2",
		"UPDATE issue SET creator = $1 WHERE creator = $2",
		"UPDATE issue_comment SET creator = $1 WHERE creator = $2",
		"UPDATE query_history SET creator = $1 WHERE creator = $2",
		"UPDATE worksheet SET creator = $1 WHERE creator = $2",
		"UPDATE worksheet_organizer SET principal = $1 WHERE principal = $2",
		"UPDATE revision SET deleter = $1 WHERE deleter = $2",
		"UPDATE release SET creator = $1 WHERE creator = $2",
		"UPDATE access_grant SET creator = $1 WHERE creator = $2",
	}
	for _, stmt := range creatorUpdates {
		if _, err := tx.ExecContext(ctx, stmt, newEmail, user.Email); err != nil {
			return nil, errors.Wrapf(err, "failed to update creator/deleter references")
		}
	}

	// 2. Update RoleGrant in issue payload.
	// The user in RoleGrant is stored as "users/{email}" within the JSON payload.
	// We use text replacement for the specific path.
	oldUserRef := common.FormatUserEmail(user.Email)
	newUserRef := common.FormatUserEmail(newEmail)

	// 'roleGrant' is the json key for role_grant field in Issue proto.
	query = qb.Q().Space(`
		UPDATE issue 
		SET payload = jsonb_set(payload, '{roleGrant,user}', to_jsonb(?::text)) 
		WHERE payload->'roleGrant'->>'user' = ?`,
		newUserRef, oldUserRef)
	sqlStr, args, err = query.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build update role grant sql")
	}
	if _, err := tx.ExecContext(ctx, sqlStr, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to update issue role grant")
	}

	// 2b. Update Approval approvers in Issue payload
	// The principal in approvers is stored as "users/{email}" within the JSON payload.
	// We need to update each approver's principal field if it matches the old user reference.
	approverSQL := `
		UPDATE issue
		SET payload = (
			SELECT jsonb_set(
				issue.payload,
				'{approval,approvers}',
				COALESCE(
					(
						SELECT jsonb_agg(
							CASE
								WHEN approver->>'principal' = $1 THEN
									jsonb_set(approver, '{principal}', to_jsonb($2::text))
								ELSE approver
							END
						)
						FROM jsonb_array_elements(issue.payload->'approval'->'approvers') AS approver
					),
					'[]'::jsonb
				)
			)
		)
		WHERE payload->'approval' ? 'approvers'
		  AND EXISTS (
			  SELECT 1
			  FROM jsonb_array_elements(payload->'approval'->'approvers') AS approver
			  WHERE approver->>'principal' = $1
		  )`

	if _, err := tx.ExecContext(ctx, approverSQL, oldUserRef, newUserRef); err != nil {
		return nil, errors.Wrapf(err, "failed to update issue approval approvers")
	}

	// 3. Update Policies
	// Update IAM policies: bindings->members array contains user references
	// Update MASKING_EXEMPTION policies: exemptions->members field contains user references
	var invalidatedPolicies []struct {
		Workspace    string
		ResourceType storepb.Policy_Resource
		Resource     string
		Type         storepb.Policy_Type
	}

	// 3a. Update IAM policy bindings
	iamPolicySQL := `
		UPDATE policy
		SET payload = (
			SELECT jsonb_set(
				policy.payload,
				'{bindings}',
				COALESCE(
					(
						SELECT jsonb_agg(
							jsonb_set(
								binding,
								'{members}',
								COALESCE(
									(
										SELECT jsonb_agg(
											CASE WHEN member = $1 THEN $2::text ELSE member END
										)
										FROM jsonb_array_elements_text(binding->'members') AS member
									),
									'[]'::jsonb
								)
							)
						)
						FROM jsonb_array_elements(policy.payload->'bindings') AS binding
					),
					'[]'::jsonb
				)
			)
		)
		WHERE type = $3
		  AND payload ? 'bindings'
		  AND EXISTS (
			  SELECT 1
			  FROM jsonb_array_elements(payload->'bindings') AS binding,
				   jsonb_array_elements_text(binding->'members') AS member
			  WHERE member = $1
		  )
		RETURNING workspace, resource_type, resource, type`

	rows, err := tx.QueryContext(ctx, iamPolicySQL, oldUserRef, newUserRef, storepb.Policy_IAM.String())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update IAM policies")
	}
	defer rows.Close()

	for rows.Next() {
		var workspace, resourceTypeStr, resource, typeStr string
		if err := rows.Scan(&workspace, &resourceTypeStr, &resource, &typeStr); err != nil {
			return nil, errors.Wrapf(err, "failed to scan updated IAM policy")
		}

		var invalidation struct {
			Workspace    string
			ResourceType storepb.Policy_Resource
			Resource     string
			Type         storepb.Policy_Type
		}
		invalidation.Workspace = workspace
		invalidation.Resource = resource

		if val, ok := storepb.Policy_Resource_value[resourceTypeStr]; ok {
			invalidation.ResourceType = storepb.Policy_Resource(val)
		}
		if val, ok := storepb.Policy_Type_value[typeStr]; ok {
			invalidation.Type = storepb.Policy_Type(val)
		}
		invalidatedPolicies = append(invalidatedPolicies, invalidation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	// 3b. Update MASKING_EXEMPTION policy exemptions
	maskingPolicySQL := `
		UPDATE policy
		SET payload = (
			SELECT jsonb_set(
				policy.payload,
				'{exemptions}',
				COALESCE(
					(
						SELECT jsonb_agg(
							CASE
								WHEN $1 = ANY(SELECT jsonb_array_elements_text(exemption->'members')) THEN
									jsonb_set(
										exemption, 
										'{members}', 
										COALESCE(
											(
												SELECT jsonb_agg(
													CASE WHEN member = $1 THEN $2::text ELSE member END
												)
												FROM jsonb_array_elements_text(exemption->'members') AS member
											),
											'[]'::jsonb
										)
									)
								ELSE exemption
							END
						)
						FROM jsonb_array_elements(policy.payload->'exemptions') AS exemption
					),
					'[]'::jsonb
				)
			)
		)
		WHERE type = $3
		  AND payload ? 'exemptions'
		  AND EXISTS (
			  SELECT 1
			  FROM jsonb_array_elements(payload->'exemptions') AS exemption,
			       jsonb_array_elements_text(exemption->'members') AS member
			  WHERE member = $1
		  )
		RETURNING workspace, resource_type, resource, type`

	rows, err = tx.QueryContext(ctx, maskingPolicySQL, oldUserRef, newUserRef, storepb.Policy_MASKING_EXEMPTION.String())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update MASKING_EXCEPTION policies")
	}
	defer rows.Close()

	for rows.Next() {
		var workspace, resourceTypeStr, resource, typeStr string
		if err := rows.Scan(&workspace, &resourceTypeStr, &resource, &typeStr); err != nil {
			return nil, errors.Wrapf(err, "failed to scan updated MASKING_EXEMPTION policy")
		}

		var invalidation struct {
			Workspace    string
			ResourceType storepb.Policy_Resource
			Resource     string
			Type         storepb.Policy_Type
		}
		invalidation.Workspace = workspace
		invalidation.Resource = resource

		if val, ok := storepb.Policy_Resource_value[resourceTypeStr]; ok {
			invalidation.ResourceType = storepb.Policy_Resource(val)
		}
		if val, ok := storepb.Policy_Type_value[typeStr]; ok {
			invalidation.Type = storepb.Policy_Type(val)
		}
		invalidatedPolicies = append(invalidatedPolicies, invalidation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	// 4. Update User Groups
	// Update user_group.payload to replace old user reference with new one in members array.
	// Members are stored as GroupMember objects with member field in "users/{email}" format.
	userGroupSQL := `
		UPDATE user_group
		SET payload = (
			SELECT jsonb_set(
				user_group.payload,
				'{members}',
				COALESCE(
					(
						SELECT jsonb_agg(
							CASE
								WHEN member->>'member' = $1 THEN
									jsonb_set(member, '{member}', to_jsonb($2::text))
								ELSE member
							END
						)
						FROM jsonb_array_elements(user_group.payload->'members') AS member
					),
					'[]'::jsonb
				)
			)
		)
		WHERE payload ? 'members'
		  AND EXISTS (
			  SELECT 1
			  FROM jsonb_array_elements(payload->'members') AS member
			  WHERE member->>'member' = $1
		  )
		RETURNING email`

	rows, err = tx.QueryContext(ctx, userGroupSQL, oldUserRef, newUserRef)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update user_group memberships")
	}
	defer rows.Close()

	var invalidatedGroupEmails []string
	for rows.Next() {
		var email sql.NullString
		if err := rows.Scan(&email); err != nil {
			return nil, errors.Wrapf(err, "failed to scan updated group")
		}
		if email.Valid {
			invalidatedGroupEmails = append(invalidatedGroupEmails, email.String)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	// 5. Update Audit Logs
	// Update audit_log.payload to replace old user reference with new one.
	// User is stored in the 'user' field in "users/{email}" format.
	query = qb.Q().Space(`
		UPDATE audit_log
		SET payload = jsonb_set(payload, '{user}', to_jsonb(?::text))
		WHERE payload->>'user' = ?`,
		newUserRef, oldUserRef)

	sqlStr, args, err = query.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build update audit_log sql")
	}

	if _, err := tx.ExecContext(ctx, sqlStr, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to update audit_log user references")
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// 6. Update caches
	s.userEmailCache.Remove(user.Email)

	// Invalidate policy cache for updated policies
	for _, p := range invalidatedPolicies {
		s.policyCache.Remove(getPolicyCacheKey(p.Workspace, p.ResourceType, p.Resource, p.Type))
		if p.Type == storepb.Policy_IAM {
			s.iamPolicyCache.Remove(getIamPolicyCacheKey(p.Workspace, p.ResourceType, p.Resource))
		}
	}

	// Purge all group caches — email change is rare and affects groups cross-workspace.
	if len(invalidatedGroupEmails) > 0 {
		s.PurgeGroupCaches()
	}

	// Re-populate user cache
	s.userEmailCache.Add(updatedUser.Email, updatedUser)

	return updatedUser, nil
}
