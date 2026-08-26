package v1

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/permission"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// Self-service credential methods. These replace the UpdateUser field masks
// and flags that used to drive password changes and MFA enrollment — same
// operations, addressed as methods rather than as a mask plus three booleans,
// so each one states who may call it and what it writes.
//
// They are auth_method=CUSTOM: the ACL interceptor enforces nothing for them,
// so every handler checks the caller itself. DisableMFA is the one method with
// an administrator path.

// ChangePassword changes the caller's own password.
func (s *UserService) ChangePassword(ctx context.Context, request *connect.Request[v1pb.ChangePasswordRequest]) (*connect.Response[v1pb.User], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	if caller.Type != storepb.PrincipalType_END_USER {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password can be mutated for end users only"))
	}
	if err := s.validatePassword(ctx, common.GetWorkspaceIDFromContext(ctx), request.Msg.NewPassword); err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(caller.PasswordHash), []byte(request.Msg.NewPassword)) == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("password cannot be the same"))
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Msg.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate password hash"))
	}

	hash := string(passwordHash)
	user, err := s.store.UpdateUser(ctx, caller, &store.UpdateUserMessage{PasswordHash: &hash})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}
	// Every session of this account, the caller's own included, has to be
	// re-established with the new password.
	if err := s.store.DeleteWebRefreshTokensByUser(ctx, user.Email); err != nil {
		slog.Error("failed to revoke refresh tokens on password change", log.BBError(err), slog.String("user", user.Email))
	}
	return s.convertUserResponse(ctx, user)
}

// StartMFAEnrollment mints a pending TOTP secret and recovery codes and hands
// them back. Any live factor is preserved: an enrollment that is abandoned, or
// superseded by a later one, changes nothing about how the account signs in.
func (s *UserService) StartMFAEnrollment(ctx context.Context, request *connect.Request[v1pb.StartMFAEnrollmentRequest]) (*connect.Response[v1pb.StartMFAEnrollmentResponse], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	tempSecret, err := generateRandSecret(caller.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate MFA secret"))
	}
	tempRecoveryCodes, err := generateRecoveryCodes(10)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate recovery codes"))
	}
	createdTime := timestamppb.Now()

	if _, err := s.store.UpdateUser(ctx, caller, &store.UpdateUserMessage{
		MFAConfig: &storepb.MFAConfig{
			OtpSecret:                caller.MFAConfig.GetOtpSecret(),
			RecoveryCodes:            caller.MFAConfig.GetRecoveryCodes(),
			TempOtpSecret:            tempSecret,
			TempRecoveryCodes:        tempRecoveryCodes,
			TempOtpSecretCreatedTime: createdTime,
		},
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}

	return connect.NewResponse(&v1pb.StartMFAEnrollmentResponse{
		OtpSecret:     tempSecret,
		RecoveryCodes: tempRecoveryCodes,
		ExpireTime:    timestamppb.New(createdTime.AsTime().Add(mfaTempSecretExpiration)),
	}), nil
}

// EnableMFA verifies an otp_code against the pending enrollment and writes
// nothing. It exists to catch a mistyped authenticator before the caller is
// handed recovery codes; ConfirmRecoveryCodes is what makes the factor live.
func (s *UserService) EnableMFA(ctx context.Context, request *connect.Request[v1pb.EnableMFARequest]) (*connect.Response[v1pb.User], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	if isMFATempSecretExpired(caller.MFAConfig) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("MFA setup has expired, please restart the enrollment"))
	}
	if !totp.Validate(request.Msg.OtpCode, caller.MFAConfig.GetTempOtpSecret()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid OTP code"))
	}
	return s.convertUserResponse(ctx, caller)
}

// DisableMFA clears the account's entire MFA config, live and pending alike, so
// a confirmation left open cannot silently re-enable it.
func (s *UserService) DisableMFA(ctx context.Context, request *connect.Request[v1pb.DisableMFARequest]) (*connect.Response[v1pb.User], error) {
	callerUser, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("failed to get caller user"))
	}
	email, err := common.GetUserEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := validateEndUserEmail(email); err != nil {
		return nil, err
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user"))
	}
	if user == nil || user.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", email))
	}

	if callerUser.ID != user.ID {
		// Same cross-user gate as UpdateUser: the principal is global, so a
		// SaaS workspace admin gets no cross-user reach here either.
		if s.profile.SaaS {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("updating other users is not allowed in SaaS mode"))
		}
		hasPermission, err := s.iamManager.CheckPermission(ctx, permission.UsersUpdate, callerUser, common.GetWorkspaceIDFromContext(ctx))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
		}
		if !hasPermission {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", permission.UsersUpdate))
		}
	}

	// A non-admin caller cannot turn MFA off while the workspace requires it,
	// self-service or admin-assisted alike — unchanged from UpdateUser.
	setting, err := s.store.GetWorkspaceProfileSetting(ctx, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace setting"))
	}
	if setting.Require_2Fa {
		isWorkspaceAdmin, err := isUserWorkspaceAdmin(ctx, s.store, callerUser, common.GetWorkspaceIDFromContext(ctx))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check user roles"))
		}
		if !isWorkspaceAdmin {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("2FA is required and cannot be disabled"))
		}
	}

	updatedUser, err := s.store.UpdateUser(ctx, user, &store.UpdateUserMessage{MFAConfig: &storepb.MFAConfig{}})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}
	return s.convertUserResponse(ctx, updatedUser)
}

// RegenerateRecoveryCodes mints a pending recovery-code set beside the live
// one. The old codes keep working until ConfirmRecoveryCodes promotes these.
func (s *UserService) RegenerateRecoveryCodes(ctx context.Context, request *connect.Request[v1pb.RegenerateRecoveryCodesRequest]) (*connect.Response[v1pb.RegenerateRecoveryCodesResponse], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	if caller.MFAConfig.GetOtpSecret() == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("MFA is not enabled"))
	}
	tempRecoveryCodes, err := generateRecoveryCodes(10)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to generate recovery codes"))
	}

	if _, err := s.store.UpdateUser(ctx, caller, &store.UpdateUserMessage{
		MFAConfig: &storepb.MFAConfig{
			OtpSecret:                caller.MFAConfig.GetOtpSecret(),
			RecoveryCodes:            caller.MFAConfig.GetRecoveryCodes(),
			TempRecoveryCodes:        tempRecoveryCodes,
			TempOtpSecretCreatedTime: timestamppb.Now(),
		},
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}

	return connect.NewResponse(&v1pb.RegenerateRecoveryCodesResponse{RecoveryCodes: tempRecoveryCodes}), nil
}

// ConfirmRecoveryCodes promotes the pending recovery codes. On a first-time
// enrollment the pending TOTP secret goes live in the same write, so the factor
// and the codes that recover it start existing together.
func (s *UserService) ConfirmRecoveryCodes(ctx context.Context, request *connect.Request[v1pb.ConfirmRecoveryCodesRequest]) (*connect.Response[v1pb.User], error) {
	caller, err := s.resolveSelfUser(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	if len(caller.MFAConfig.GetTempRecoveryCodes()) == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("no pending recovery codes to confirm"))
	}

	mfaConfig := &storepb.MFAConfig{
		OtpSecret:     caller.MFAConfig.GetOtpSecret(),
		RecoveryCodes: caller.MFAConfig.GetTempRecoveryCodes(),
	}
	// A pending secret means an enrollment — the account's first factor, or a
	// replacement for the one it already has — and either way that secret is
	// the one the caller just proved they can generate codes from, so it is
	// the one that goes live. Only RegenerateRecoveryCodes leaves no pending
	// secret, and only there does the live factor survive untouched.
	if pendingSecret := caller.MFAConfig.GetTempOtpSecret(); pendingSecret != "" {
		if isMFATempSecretExpired(caller.MFAConfig) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("MFA setup has expired, please restart the enrollment"))
		}
		mfaConfig.OtpSecret = pendingSecret
	} else if mfaConfig.OtpSecret == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("no MFA enrollment in progress, call StartMFAEnrollment first"))
	}

	user, err := s.store.UpdateUser(ctx, caller, &store.UpdateUserMessage{MFAConfig: mfaConfig})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update user"))
	}
	return s.convertUserResponse(ctx, user)
}

// resolveSelfUser parses name and requires it to be the caller's own account.
// Nothing upstream enforces this for CUSTOM methods, so every self-service
// method goes through here.
func (*UserService) resolveSelfUser(ctx context.Context, name string) (*store.UserMessage, error) {
	caller, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("failed to get caller user"))
	}
	email, err := common.GetUserEmail(name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := validateEndUserEmail(email); err != nil {
		return nil, err
	}
	if caller.Email != email {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("name must be the caller's own account"))
	}
	return caller, nil
}

func (s *UserService) convertUserResponse(ctx context.Context, user *store.UserMessage) (*connect.Response[v1pb.User], error) {
	v1User, err := convertToUser(ctx, s.iamManager, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
	}
	return connect.NewResponse(v1User), nil
}
