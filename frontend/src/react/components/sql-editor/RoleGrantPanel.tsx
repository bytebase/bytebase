import { create } from "@bufbuild/protobuf";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { roleHasDatabaseLimitation } from "@/components/ProjectMember/utils";
import { issueServiceClientConnect } from "@/connect";
import { DatabaseResourceSelector } from "@/react/components/DatabaseResourceSelector";
import { IssueLabelSelect } from "@/react/components/IssueLabelSelect";
import { RoleSelect } from "@/react/components/RoleSelect";
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
  useProjectV1Store,
  useRoleStore,
  useSettingV1Store,
} from "@/store";
import { type DatabaseResource, PresetRoleType } from "@/types";
import {
  CreateIssueRequestSchema,
  Issue_Type,
  IssueSchema,
  RoleGrantSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  displayRoleTitle,
  extractIssueUID,
  extractProjectResourceName,
  formatIssueTitle,
} from "@/utils";
import { buildConditionExpr } from "@/utils/issue/cel";

interface Props {
  readonly projectName: string;
  readonly databaseResources: DatabaseResource[];
  readonly role: string;
  readonly requiredPermissions: string[];
  readonly onClose: () => void;
}

interface RoleGrantPanelInnerProps {
  readonly stableProjectName: string;
  readonly databaseResources: DatabaseResource[];
  readonly role: string;
  readonly requiredPermissions: string[];
  readonly onClose: () => void;
}

function RoleGrantPanelInner({
  stableProjectName,
  databaseResources,
  role,
  requiredPermissions,
  onClose,
}: RoleGrantPanelInnerProps) {
  const { t } = useTranslation();

  const currentUserRef = useCurrentUserV1();
  const currentUser = useVueState(() => currentUserRef.value);

  const projectStore = useProjectV1Store();
  const project = useVueState(() =>
    projectStore.getProjectByName(stableProjectName)
  );

  const roleStore = useRoleStore();
  const selectedRole = useVueState(() => roleStore.getRoleByName(role));

  const settingStore = useSettingV1Store();
  const maximumRoleExpirationDays = useVueState(() => {
    if (role === PresetRoleType.PROJECT_OWNER) return undefined;
    const seconds = settingStore.workspaceProfile.maximumRoleExpiration?.seconds;
    if (!seconds) return undefined;
    return Math.floor(Number(seconds) / (60 * 60 * 24));
  });

  const showDatabaseSelector = roleHasDatabaseLimitation(role);

  const [dbResources, setDbResources] =
    useState<DatabaseResource[]>(databaseResources);
  const [reason, setReason] = useState("");
  const [expirationTimestamp, setExpirationTimestamp] = useState<
    string | undefined
  >(undefined);
  const [labels, setLabels] = useState<string[]>([]);
  const [submitting, setSubmitting] = useState(false);

  const minDatetime = dayjs().format("YYYY-MM-DDTHH:mm");
  const maxDatetime = maximumRoleExpirationDays
    ? dayjs().add(maximumRoleExpirationDays, "days").format("YYYY-MM-DDTHH:mm")
    : undefined;

  const reasonRequired = useMemo(
    () => project?.enforceIssueTitle ?? false,
    [project]
  );

  const labelsMisconfigured = useMemo(
    () => !!(project?.forceIssueLabels && project.issueLabels.length === 0),
    [project]
  );

  const expirationIsInPast =
    !!expirationTimestamp &&
    dayjs(expirationTimestamp).unix() <= dayjs().unix();

  const expirationExceedsMax =
    !!expirationTimestamp &&
    !!maximumRoleExpirationDays &&
    dayjs(expirationTimestamp).isAfter(
      dayjs().add(maximumRoleExpirationDays, "days")
    );

  // When the workspace has a maximumRoleExpiration policy, the user must
  // pick an expiration — otherwise the backend treats missing expiration as
  // effectively unbounded (math.MaxInt32 days) and bypasses the cap.
  const expirationRequired = maximumRoleExpirationDays !== undefined;

  const canSubmit = useMemo(() => {
    if (submitting) return false;
    if (reasonRequired && !reason.trim()) return false;
    if (showDatabaseSelector && dbResources.length === 0) return false;
    if (expirationRequired && !expirationTimestamp) return false;
    if (expirationIsInPast) return false;
    if (expirationExceedsMax) return false;
    if (project?.forceIssueLabels && labels.length === 0) return false;
    if (labelsMisconfigured) return false;
    return true;
  }, [
    submitting,
    reasonRequired,
    reason,
    showDatabaseSelector,
    dbResources,
    expirationRequired,
    expirationTimestamp,
    expirationIsInPast,
    expirationExceedsMax,
    project,
    labels,
    labelsMisconfigured,
  ]);

  const handleSubmit = async () => {
    if (!canSubmit) return;
    setSubmitting(true);
    try {
      const trimmedReason = reason.trim();
      const expirationTimestampInMS = expirationTimestamp
        ? dayjs(expirationTimestamp).valueOf()
        : undefined;

      const condition = buildConditionExpr({
        role,
        description: trimmedReason,
        expirationTimestampInMS,
        databaseResources: dbResources.length > 0 ? dbResources : undefined,
      });

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

      const titleDatabaseNames =
        dbResources.length > 0
          ? [...new Set(dbResources.map((r) => r.databaseFullName))]
          : undefined;

      const title = project?.enforceIssueTitle
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
          parent: stableProjectName,
          issue: newIssue,
        })
      );

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.created"),
      });

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
      // Error notification pushed by client middleware.
    } finally {
      setSubmitting(false);
    }
  };

  const issueLabels = project?.issueLabels ?? [];
  const forceIssueLabels = !!project?.forceIssueLabels;
  const showLabelSelect = issueLabels.length > 0 || forceIssueLabels;

  return (
    <>
      <SheetHeader>
        <SheetTitle>{t("issue.title.request-role")}</SheetTitle>
      </SheetHeader>
      <SheetBody>
        <div className="flex flex-col gap-y-4">
          {requiredPermissions.length > 0 && (
            <div className="flex flex-col gap-y-1">
              <label className="text-sm font-medium text-control">
                {t("common.required-permission")}
              </label>
              <ul className="list-disc pl-4 text-sm text-control-light">
                {requiredPermissions.map((p) => (
                  <li key={p}>{p}</li>
                ))}
              </ul>
            </div>
          )}

          <div className="flex flex-col gap-y-1">
            <label className="text-sm font-medium text-control">
              {t("common.role.self")}
            </label>
            <RoleSelect
              value={[role]}
              onChange={() => {}}
              multiple={false}
              disabled={true}
              scope="project"
            />
          </div>

          {selectedRole && selectedRole.permissions.length > 0 && (
            <div className="flex flex-col gap-y-1">
              <label className="text-sm font-medium text-control">
                {t("common.permissions")} ({selectedRole.permissions.length})
              </label>
              <div className="max-h-[10em] overflow-auto border border-control-border rounded-sm p-2">
                <ul className="list-none">
                  {selectedRole.permissions.map((p) => (
                    <li key={p} className="text-sm leading-5">
                      {p}
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          )}

          <div className="flex flex-col gap-y-1">
            <label className="text-sm font-medium text-control">
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

          {showDatabaseSelector && (
            <div className="flex flex-col gap-y-1">
              <label className="text-sm font-medium text-control">
                {t("common.databases")}
                <span className="text-error ml-0.5">*</span>
              </label>
              <DatabaseResourceSelector
                projectName={stableProjectName}
                value={dbResources}
                onChange={setDbResources}
              />
            </div>
          )}

          <div className="flex flex-col gap-y-1">
            <label className="text-sm font-medium text-control">
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
          </div>

          {showLabelSelect && (
            <IssueLabelSelect
              labels={issueLabels}
              selected={labels}
              required={forceIssueLabels}
              onChange={setLabels}
            />
          )}
        </div>
      </SheetBody>
      <SheetFooter>
        <Button variant="outline" onClick={onClose} disabled={submitting}>
          {t("common.cancel")}
        </Button>
        <Button disabled={!canSubmit} onClick={() => void handleSubmit()}>
          {t("common.submit")}
        </Button>
      </SheetFooter>
    </>
  );
}

export function RoleGrantPanel({
  projectName,
  databaseResources,
  role,
  requiredPermissions,
  onClose,
}: Props) {
  return (
    <Sheet open={true} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <RoleGrantPanelInner
          key={projectName}
          stableProjectName={projectName}
          databaseResources={databaseResources}
          role={role}
          requiredPermissions={requiredPermissions}
          onClose={onClose}
        />
      </SheetContent>
    </Sheet>
  );
}
