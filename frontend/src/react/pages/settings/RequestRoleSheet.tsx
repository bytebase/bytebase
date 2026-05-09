import { create } from "@bufbuild/protobuf";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  getRoleEnvironmentLimitationKind,
  roleHasDatabaseLimitation,
} from "@/components/ProjectMember/utils";
import { issueServiceClientConnect } from "@/connect";
import type { ConditionGroupExpr, Factor, Operator } from "@/plugins/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import { DatabaseResourceSelector as DatabaseResourceSelectorComponent } from "@/react/components/DatabaseResourceSelector";
import { EnvironmentMultiSelect } from "@/react/components/EnvironmentMultiSelect";
import type { OptionConfig } from "@/react/components/ExprEditor";
import { ExprEditor } from "@/react/components/ExprEditor";
import { IssueLabelSelect } from "@/react/components/IssueLabelSelect";
import { RoleSelect } from "@/react/components/RoleSelect";
import { DDLWarningCallout } from "@/react/components/role-grant/DDLWarningCallout";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { ExpirationPicker } from "@/react/components/ui/expiration-picker";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Textarea } from "@/react/components/ui/textarea";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useCurrentUserV1,
  useRoleStore,
  useSettingV1Store,
} from "@/store";
import type { Permission } from "@/types";
import { type DatabaseResource, PresetRoleType } from "@/types";
import { ExprSchema as ConditionExprSchema } from "@/types/proto-es/google/type/expr_pb";
import {
  CreateIssueRequestSchema,
  Issue_Type,
  IssueSchema,
  RoleGrantSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import {
  batchConvertParsedExprToCELString,
  displayRoleTitle,
  extractIssueUID,
  extractProjectResourceName,
  formatIssueTitle,
  getDatabaseNameOptionConfig,
  normalizeTitle,
} from "@/utils";
import {
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";
import {
  buildConditionExpr,
  stringifyConditionExpression,
} from "@/utils/issue/cel";
import type { DatabaseMode } from "./types";

export interface RequestRoleSheetProps {
  open: boolean;
  project: Project;
  requiredPermissions?: Permission[];
  onClose: () => void;
}

const EMPTY_REQUIRED_PERMISSIONS: Permission[] = [];

// i18n key lookup for the three DatabaseMode radio labels. Kept as a flat
// function to avoid nested ternaries in the render tree.
const databaseModeLabelKey = (mode: DatabaseMode): string => {
  switch (mode) {
    case "ALL":
      return "issue.role-grant.all-databases";
    case "EXPRESSION":
      return "issue.role-grant.use-cel";
    case "SELECT":
      return "issue.role-grant.manually-select";
  }
};

export function RequestRoleSheet(props: Readonly<RequestRoleSheetProps>) {
  const { open, project, onClose } = props;
  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        {/* Base UI's Dialog.Portal unmounts after the close animation,
            so the inner form mounts/unmounts with the Sheet's lifecycle
            without needing an explicit {open && ...} guard. The `key`
            forces a fresh mount if the project switches while open. */}
        <RequestRoleForm key={project.name} {...props} />
      </SheetContent>
    </Sheet>
  );
}

function RequestRoleForm({
  project,
  requiredPermissions = EMPTY_REQUIRED_PERMISSIONS,
  onClose,
}: Readonly<Omit<RequestRoleSheetProps, "open">>) {
  const { t } = useTranslation();
  // Call the Pinia store accessor at the top level of the component so
  // SonarCloud's React-hook-rule doesn't flag the `use*` call inside
  // a subscription getter. `useCurrentUserV1()` returns a cached Pinia
  // computed ref, so calling it once and reading `.value` inside the
  // getter produces identical reactive semantics.
  const currentUserRef = useCurrentUserV1();
  const currentUser = useVueState(() => currentUserRef.value);
  const [role, setRole] = useState("");
  const [reason, setReason] = useState("");
  const [expirationTimestamp, setExpirationTimestamp] = useState<
    string | undefined
  >(undefined);
  const [labels, setLabels] = useState<string[]>([]);
  // Role-scope fields — only rendered for roles that require them, but kept
  // in state so switching roles back and forth preserves user input.
  const [databaseMode, setDatabaseMode] = useState<DatabaseMode>("ALL");
  const [databaseResources, setDatabaseResources] = useState<
    DatabaseResource[]
  >([]);
  // CEL expression for EXPRESSION mode is held as a structured group so it
  // can render in ExprEditor (matching the old Vue DatabaseResourceForm).
  const [exprGroup, setExprGroup] = useState<ConditionGroupExpr>(() =>
    wrapAsGroup(emptySimpleExpr())
  );
  const [environments, setEnvironments] = useState<string[]>([]);
  const [submitting, setSubmitting] = useState(false);

  const settingStore = useSettingV1Store();
  const roleStore = useRoleStore();
  const selectedRole = useVueState(() =>
    role ? roleStore.getRoleByName(role) : undefined
  );
  const requiredPermissionList = useMemo(
    () => [...new Set(requiredPermissions)],
    [requiredPermissions]
  );
  const roleMatchesRequiredPermissions = (candidate: Role | undefined) => {
    if (requiredPermissionList.length === 0) {
      return true;
    }
    if (!candidate) {
      return false;
    }
    return requiredPermissionList.every((permission) =>
      candidate.permissions.includes(permission)
    );
  };
  const selectedRoleMatchesRequiredPermissions =
    roleMatchesRequiredPermissions(selectedRole);

  // Workspace-configured maximum role expiration, in days. Matches the old
  // Vue ExpirationSelector: PROJECT_OWNER grants are exempted (project
  // owners can request unbounded expirations), otherwise the workspace cap
  // applies. Returns undefined when no cap is set.
  const maximumRoleExpirationDays = useVueState(() => {
    if (role === PresetRoleType.PROJECT_OWNER) return undefined;
    const seconds =
      settingStore.workspaceProfile.maximumRoleExpiration?.seconds;
    if (!seconds) return undefined;
    return Math.floor(Number(seconds) / (60 * 60 * 24));
  });

  // ExprEditor factor/option config — mirrors the old Vue
  // DatabaseResourceForm config for role-grant requests.
  const factorList = useMemo<Factor[]>(
    () => [
      CEL_ATTRIBUTE_RESOURCE_DATABASE,
      CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
      CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
    ],
    []
  );
  const factorOperatorOverrideMap = useMemo(
    () =>
      new Map<Factor, Operator[]>([
        [CEL_ATTRIBUTE_RESOURCE_DATABASE, ["_==_", "@in"]],
        [CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME, ["_==_"]],
        [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME, ["_==_", "@in"]],
      ]),
    []
  );
  const factorOptionConfigMap = useMemo(
    () =>
      factorList.reduce((map, factor) => {
        if (factor === CEL_ATTRIBUTE_RESOURCE_DATABASE) {
          map.set(factor, getDatabaseNameOptionConfig(project.name));
        } else {
          map.set(factor, { options: [] });
        }
        return map;
      }, new Map<Factor, OptionConfig>()),
    [factorList, project.name]
  );

  // Bind the datetime-local picker's min to the current minute so users
  // can't silently pick a time earlier today and submit a zero-duration
  // role grant.
  const minDatetime = dayjs().format("YYYY-MM-DDTHH:mm");
  const maxDatetime = maximumRoleExpirationDays
    ? dayjs().add(maximumRoleExpirationDays, "days").format("YYYY-MM-DDTHH:mm")
    : undefined;

  const expirationIsInPast =
    !!expirationTimestamp &&
    dayjs(expirationTimestamp).unix() <= dayjs().unix();

  // Workspace policy may cap role expiration; block submit when the picked
  // timestamp would exceed that cap (matches Vue ExpirationSelector which
  // disables over-cap dates in its picker).
  const expirationExceedsMax =
    !!expirationTimestamp &&
    !!maximumRoleExpirationDays &&
    dayjs(expirationTimestamp).isAfter(
      dayjs().add(maximumRoleExpirationDays, "days")
    );

  const labelsMisconfigured =
    project.forceIssueLabels && project.issueLabels.length === 0;

  // Match the old Vue AddProjectMemberForm behavior: reason is only required
  // when the project enforces issue titles (where the reason becomes the
  // title). Otherwise the title is auto-generated and the reason is optional.
  const reasonRequired = project.enforceIssueTitle;

  // SQL-permission roles need a database scope (and sometimes an environment
  // scope). Submitting without one produces a project-wide binding which is
  // broader than the user typically intends.
  const showDatabases = !!role && roleHasDatabaseLimitation(role);
  const envKind = role ? getRoleEnvironmentLimitationKind(role) : undefined;

  const databaseScopeComplete =
    !showDatabases ||
    databaseMode === "ALL" ||
    (databaseMode === "SELECT" && databaseResources.length > 0) ||
    (databaseMode === "EXPRESSION" && validateSimpleExpr(exprGroup));

  // When the workspace enforces a maximum role expiration, the user MUST
  // pick an expiration — leaving it empty sends `roleGrant.expiration` as
  // `undefined`, which the backend's approval runner treats as effectively
  // unbounded (math.MaxInt32 days), bypassing the cap entirely.
  const expirationRequired = maximumRoleExpirationDays !== undefined;

  const canSubmit =
    !submitting &&
    !!role &&
    (!reasonRequired || normalizeTitle(reason).length > 0) &&
    (!expirationRequired || !!expirationTimestamp) &&
    !expirationIsInPast &&
    !expirationExceedsMax &&
    !labelsMisconfigured &&
    databaseScopeComplete &&
    selectedRoleMatchesRequiredPermissions &&
    (!project.forceIssueLabels || labels.length > 0);

  const handleSubmit = async () => {
    if (!canSubmit) return;
    setSubmitting(true);
    try {
      const trimmedReason = normalizeTitle(reason);
      const expirationTimestampInMS = expirationTimestamp
        ? dayjs(expirationTimestamp).valueOf()
        : undefined;

      // Only pass scope filters when the selected role actually requires
      // them, matching EditMemberRoleDrawer's submit logic.
      const scopedDatabaseResources =
        showDatabases &&
        databaseMode === "SELECT" &&
        databaseResources.length > 0
          ? databaseResources
          : undefined;
      // EnvLimitationKind union has no falsy members, so the truthy check
      // is equivalent to !== undefined.
      const scopedEnvironments = envKind ? environments : undefined;

      // The backend uses two fields on the RoleGrant message:
      //   1. `condition.expression` — CEL evaluated by
      //      UpdateProjectPolicyFromRoleGrantIssue to gate the resulting
      //      project IAM binding (access control).
      //   2. `expiration` — Duration consumed by
      //      backend/runner/approval/runner.go (request.expiration_days) to
      //      route the approval chain; unset defaults to math.MaxInt32 days.
      // Both must be populated — the condition controls what the grant
      // allows, the expiration controls which approvers review it.
      let condition;
      if (showDatabases && databaseMode === "EXPRESSION") {
        // Convert the structured ExprEditor group to a raw CEL string and
        // combine it with any environment/expiration clauses. Mirrors
        // ProjectMaskingExemptionCreatePage's EXPRESSION-mode submit path.
        const parsedExpr = await buildCELExpr(exprGroup);
        if (!parsedExpr) {
          throw new Error("failed to build CEL expression");
        }
        const [exprString] = await batchConvertParsedExprToCELString([
          parsedExpr,
        ]);
        // `batchConvertParsedExprToCELString` swallows deparse errors and
        // returns an empty string on failure. Submitting anyway would
        // silently drop the database scope filter and produce a broader
        // grant than the user requested — refuse instead and surface an
        // error so the user can retry.
        if (!exprString?.trim()) {
          pushNotification({
            module: "bytebase",
            style: "CRITICAL",
            title: t("project.members.request-role.failed-to-build-expression"),
          });
          return;
        }
        const extraParts = stringifyConditionExpression({
          expirationTimestampInMS,
          environments: scopedEnvironments,
        });
        const fullExpression = extraParts
          ? `(${exprString}) && ${extraParts}`
          : exprString;
        condition = create(ConditionExprSchema, {
          expression: fullExpression,
          description: trimmedReason,
        });
      } else {
        condition = buildConditionExpr({
          role,
          description: trimmedReason,
          expirationTimestampInMS,
          databaseResources: scopedDatabaseResources,
          environments: scopedEnvironments,
        });
      }

      const expiration = expirationTimestamp
        ? create(DurationSchema, {
            seconds: BigInt(
              Math.max(0, dayjs(expirationTimestamp).unix() - dayjs().unix())
            ),
          })
        : undefined;

      const roleGrant = create(RoleGrantSchema, {
        role,
        user: `users/${currentUser.email}`,
        condition,
        expiration,
      });

      // Collect the scoped database names so the auto-generated title shows
      // `[db1]` / `[N databases]` instead of defaulting to `[All databases]`
      // — matches the old Vue RoleGrantPanel behavior. EXPRESSION mode
      // currently falls through to `[All databases]` since parsing CEL back
      // into concrete database names would require an async round-trip.
      const titleDatabaseNames =
        scopedDatabaseResources && scopedDatabaseResources.length > 0
          ? [...new Set(scopedDatabaseResources.map((r) => r.databaseFullName))]
          : undefined;

      // When the project enforces issue titles, the user-provided reason is
      // treated as the title (matching the old Vue RoleGrantPanel behavior).
      const title = project.enforceIssueTitle
        ? `[${t("issue.title.request-role")}] ${trimmedReason}`
        : formatIssueTitle(
            t("issue.title.request-specific-role", {
              role: displayRoleTitle(role),
            }),
            titleDatabaseNames
          );

      const newIssue = create(IssueSchema, {
        title,
        description: trimmedReason,
        type: Issue_Type.ROLE_GRANT,
        roleGrant,
        labels,
      });
      const response = await issueServiceClientConnect.createIssue(
        create(CreateIssueRequestSchema, {
          parent: project.name,
          issue: newIssue,
        })
      );
      const route = router.resolve({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(response.name),
          issueId: extractIssueUID(response.name),
        },
      });
      window.open(route.fullPath, "_blank", "noopener,noreferrer");
      onClose();
    } catch {
      // Error notification is pushed by the client middleware.
    } finally {
      setSubmitting(false);
    }
  };

  const showLabelSelect =
    project.issueLabels.length > 0 || project.forceIssueLabels;

  return (
    <>
      <SheetHeader>
        <SheetTitle>{t("issue.title.request-role")}</SheetTitle>
      </SheetHeader>
      <SheetBody>
        <div className="flex flex-col gap-y-4">
          {labelsMisconfigured && (
            <Alert
              variant="warning"
              title={t(
                "project.members.request-role.labels-misconfigured.title"
              )}
              description={t(
                "project.members.request-role.labels-misconfigured.description"
              )}
            />
          )}
          {requiredPermissionList.length > 0 && (
            <Alert
              title={t("common.required-permission")}
              description={
                <ul className="list-disc pl-4">
                  {requiredPermissionList.map((permission) => (
                    <li key={permission}>{permission}</li>
                  ))}
                </ul>
              }
            />
          )}
          <div className="flex flex-col gap-y-1">
            <label className="text-sm font-medium">
              {t("common.role.self")}
              <span className="text-error ml-0.5">*</span>
            </label>
            <RoleSelect
              scope="project"
              multiple={false}
              value={role ? [role] : []}
              onChange={(roles) => {
                setRole(roles[0] ?? "");
                // Reset scope when switching roles since DB/env fields only
                // apply to some roles and the new role may not use them.
                setDatabaseMode("ALL");
                setDatabaseResources([]);
                setExprGroup(wrapAsGroup(emptySimpleExpr()));
                setEnvironments([]);
              }}
              filterRole={roleMatchesRequiredPermissions}
            />
            {!!role && !selectedRoleMatchesRequiredPermissions && (
              <p className="text-xs text-error">
                {t("common.missing-required-permission", {
                  permissions: requiredPermissionList.join(", "),
                })}
              </p>
            )}
          </div>
          <div className="flex flex-col gap-y-1">
            <label className="text-sm font-medium">
              {t("common.reason")}
              {reasonRequired && <span className="text-error ml-0.5">*</span>}
            </label>
            <Textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder={t("common.reason")}
              rows={3}
            />
          </div>
          {showDatabases && (
            <div className="flex flex-col gap-y-2">
              <label className="text-sm font-medium">
                {t("common.databases")}
                <span className="text-error ml-0.5">*</span>
              </label>
              <div className="flex items-center gap-x-4">
                {(["ALL", "EXPRESSION", "SELECT"] as DatabaseMode[]).map(
                  (m) => (
                    <label
                      key={m}
                      className="flex items-center gap-x-2 text-sm cursor-pointer"
                    >
                      <input
                        type="radio"
                        name="request-role-db-mode"
                        checked={databaseMode === m}
                        onChange={() => {
                          setDatabaseMode(m);
                          setDatabaseResources([]);
                          setExprGroup(wrapAsGroup(emptySimpleExpr()));
                        }}
                      />
                      {t(databaseModeLabelKey(m))}
                    </label>
                  )
                )}
              </div>
              {databaseMode === "EXPRESSION" && (
                <ExprEditor
                  expr={exprGroup}
                  factorList={factorList}
                  optionConfigMap={factorOptionConfigMap}
                  factorOperatorOverrideMap={factorOperatorOverrideMap}
                  onUpdate={setExprGroup}
                />
              )}
              {databaseMode === "SELECT" && (
                <DatabaseResourceSelectorComponent
                  projectName={project.name}
                  value={databaseResources}
                  onChange={setDatabaseResources}
                />
              )}
            </div>
          )}
          {envKind && (
            <div className="flex flex-col gap-y-2">
              <label className="text-sm font-medium">
                {t("common.environments")}
              </label>
              <DDLWarningCallout type="drawer" kind={envKind} />
              <EnvironmentMultiSelect
                value={environments}
                onChange={setEnvironments}
              />
            </div>
          )}
          <div className="flex flex-col gap-y-1">
            <label className="text-sm font-medium">
              {t("common.expiration")}
              {expirationRequired && (
                <span className="text-error ml-0.5">*</span>
              )}
            </label>
            <ExpirationPicker
              value={expirationTimestamp}
              onChange={setExpirationTimestamp}
              minDate={minDatetime}
              maxDate={maxDatetime}
            />
            {maximumRoleExpirationDays !== undefined && (
              <p className="text-xs text-control-light">
                {t("project.members.request-role.max-expiration-hint", {
                  days: maximumRoleExpirationDays,
                })}
              </p>
            )}
            {expirationIsInPast && (
              <p className="text-xs text-error">
                {t("project.members.request-role.expiration-must-be-future")}
              </p>
            )}
            {expirationExceedsMax && (
              <p className="text-xs text-error">
                {t("project.members.request-role.expiration-exceeds-max", {
                  days: maximumRoleExpirationDays,
                })}
              </p>
            )}
          </div>
          {showLabelSelect && (
            <IssueLabelSelect
              labels={project.issueLabels}
              selected={labels}
              required={project.forceIssueLabels}
              onChange={setLabels}
            />
          )}
        </div>
      </SheetBody>
      <SheetFooter>
        <Button variant="outline" onClick={onClose} disabled={submitting}>
          {t("common.cancel")}
        </Button>
        <Button disabled={!canSubmit} onClick={handleSubmit}>
          {t("common.submit")}
        </Button>
      </SheetFooter>
    </>
  );
}
