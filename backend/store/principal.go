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

// SystemBotUser is the static system bot user.
var SystemBotUser = &UserMessage{
	ID:    common.SystemBotID,
	Name:  "Bytebase",
	Email: "support@bytebase.com",
	Type:  storepb.PrincipalType_SYSTEM_BOT,
}

// FindUserMessage is the message for finding users.
type FindUserMessage struct {
	ID          *int
	Email       *string
	ShowDeleted bool
	Type        *storepb.PrincipalType
	Limit       *int
	Offset      *int
	FilterQ     *qb.Query
	ProjectID   *string
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

// UserMessage is the message for an user.
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

type UserStat struct {
	Type    storepb.PrincipalType
	Deleted bool
	Count   int
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

func (s *Store) StatUsers(ctx context.Context) ([]*UserStat, error) {
	q := qb.Q().Space(`
		SELECT
			COUNT(*),
			type,
			deleted
		FROM principal
		GROUP BY type, deleted
	`)
	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*UserStat

	for rows.Next() {
		var stat UserStat
		var typeString string
		if err := rows.Scan(
			&stat.Count,
			&typeString,
			&stat.Deleted,
		); err != nil {
			return nil, err
		}
		if typeValue, ok := storepb.PrincipalType_value[typeString]; ok {
			stat.Type = storepb.PrincipalType(typeValue)
		} else {
			return nil, errors.Errorf("invalid principal type string: %s", typeString)
		}
		stats = append(stats, &stat)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to scan rows")
	}

	return stats, nil
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
			WHERE ((resource_type = ? AND resource = ?) OR resource_type = ?) AND type = ?
		),
		project_members AS (
			SELECT ARRAY_AGG(member) AS members FROM all_members WHERE role NOT LIKE 'roles/workspace%'
		)`, storepb.Policy_PROJECT.String(), "projects/"+*v, storepb.Policy_WORKSPACE.String(), storepb.Policy_IAM.String())
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
	if v := find.Type; v != nil {
		where.And("principal.type = ?", v.String())
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
			principal.type,
			principal.password_hash,
			principal.mfa_config,
			principal.phone,
			principal.profile,
			principal.created_at
		FROM ?
		WHERE ?
		ORDER BY type DESC, created_at ASC
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
	rows, err := txn.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userMessage UserMessage
		var mfaConfigBytes []byte
		var profileBytes []byte
		var typeString string
		if err := rows.Scan(
			&userMessage.ID,
			&userMessage.MemberDeleted,
			&userMessage.Email,
			&userMessage.Name,
			&typeString,
			&userMessage.PasswordHash,
			&mfaConfigBytes,
			&userMessage.Phone,
			&profileBytes,
			&userMessage.CreatedAt,
		); err != nil {
			return nil, err
		}
		if typeValue, ok := storepb.PrincipalType_value[typeString]; ok {
			userMessage.Type = storepb.PrincipalType(typeValue)
		} else {
			return nil, errors.Errorf("invalid principal type string: %s", typeString)
		}

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
	var typeString string
	if err := tx.QueryRowContext(ctx, sqlStr, args...).Scan(
		&user.ID,
		&user.MemberDeleted,
		&user.Email,
		&user.Name,
		&typeString,
		&user.PasswordHash,
		&mfaConfigBytes,
		&user.Phone,
		&profileBytes,
		&user.CreatedAt,
	); err != nil {
		return nil, err
	}

	if typeValue, ok := storepb.PrincipalType_value[typeString]; ok {
		user.Type = storepb.PrincipalType(typeValue)
	} else {
		return nil, errors.Errorf("invalid principal type string: %s", typeString)
	}

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

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

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
			type,
			password_hash,
			phone,
			profile
		)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id, created_at
	`, create.Email, create.Name, create.Type.String(), create.PasswordHash, create.Phone, profileBytes)

	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var userID int
	if err := tx.QueryRowContext(ctx, sql, args...).Scan(&userID, &create.CreatedAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	user := &UserMessage{
		ID:           userID,
		Email:        create.Email,
		Name:         create.Name,
		Type:         create.Type,
		PasswordHash: create.PasswordHash,
		Phone:        create.Phone,
		CreatedAt:    create.CreatedAt,
		Profile:      create.Profile,
		MFAConfig:    &storepb.MFAConfig{},
	}
	s.userEmailCache.Add(user.Email, user)
	return user, nil
}

// UpdateUser updates a user.
func (s *Store) UpdateUser(ctx context.Context, currentUser *UserMessage, patch *UpdateUserMessage) (*UserMessage, error) {
	if currentUser.ID == common.SystemBotID {
		return nil, errors.Errorf("cannot update system bot")
	}

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
		RETURNING id, deleted, email, name, type, password_hash, mfa_config, phone, profile, created_at`,
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
	// Because we have ON UPDATE CASCADE on foreign keys (issue.creator, etc.),
	// this will automatically update the creator field in:
	// - issue
	// - issue_comment
	// - simple table references (plan, pipeline, task_run, etc.)
	query := qb.Q().Space(`UPDATE principal SET email = ? WHERE id = ?
		RETURNING id, deleted, email, name, type, password_hash, mfa_config, phone, profile, created_at`,
		newEmail, user.ID)
	sqlStr, args, err := query.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build update principal sql")
	}

	updatedUser, err := scanPrincipalRow(ctx, tx, sqlStr, args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update principal email")
	}

	// 2. Update GrantRequest in Issue payload
	// The user in GrantRequest is stored as "users/{email}" within the JSON payload.
	// We use text replacement for the specific path.
	oldUserRef := common.FormatUserEmail(user.Email)
	newUserRef := common.FormatUserEmail(newEmail)

	// 'grantRequest' is the json key for grant_request field in Issue proto
	query = qb.Q().Space(`
		UPDATE issue 
		SET payload = jsonb_set(payload, '{grantRequest,user}', to_jsonb(?::text)) 
		WHERE payload->'grantRequest'->>'user' = ?`,
		newUserRef, oldUserRef)
	sqlStr, args, err = query.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build update grant request sql")
	}
	if _, err := tx.ExecContext(ctx, sqlStr, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to update issue grant request")
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
		RETURNING resource_type, resource, type`

	rows, err := tx.QueryContext(ctx, iamPolicySQL, oldUserRef, newUserRef, storepb.Policy_IAM.String())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update IAM policies")
	}
	defer rows.Close()

	for rows.Next() {
		var resourceTypeStr, resource, typeStr string
		if err := rows.Scan(&resourceTypeStr, &resource, &typeStr); err != nil {
			return nil, errors.Wrapf(err, "failed to scan updated IAM policy")
		}

		var invalidation struct {
			ResourceType storepb.Policy_Resource
			Resource     string
			Type         storepb.Policy_Type
		}
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
		RETURNING resource_type, resource, type`

	rows, err = tx.QueryContext(ctx, maskingPolicySQL, oldUserRef, newUserRef, storepb.Policy_MASKING_EXEMPTION.String())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update MASKING_EXCEPTION policies")
	}
	defer rows.Close()

	for rows.Next() {
		var resourceTypeStr, resource, typeStr string
		if err := rows.Scan(&resourceTypeStr, &resource, &typeStr); err != nil {
			return nil, errors.Wrapf(err, "failed to scan updated MASKING_EXEMPTION policy")
		}

		var invalidation struct {
			ResourceType storepb.Policy_Resource
			Resource     string
			Type         storepb.Policy_Type
		}
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
		s.policyCache.Remove(getPolicyCacheKey(p.ResourceType, p.Resource, p.Type))
	}

	// Invalidate group cache for updated groups
	for _, email := range invalidatedGroupEmails {
		s.groupCache.Remove(email)
	}

	// Re-populate user cache
	s.userEmailCache.Add(updatedUser.Email, updatedUser)

	return updatedUser, nil
}
