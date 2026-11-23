import { Check, Pencil, Plus, Trash2, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import {
  WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
  WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
} from "@/app/router/handles";
import { HighlightLabelText } from "@/components/HighlightLabelText";
import { LearnMoreLink } from "@/components/LearnMoreLink";
import { ResourceLink } from "@/components/sql-review/ResourceLink";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FormSection } from "@/components/ui/form";
import { SearchInput } from "@/components/ui/search-input";
import { StickyActionFooter } from "@/components/ui/sticky-action-footer";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tooltip } from "@/components/ui/tooltip";
import {
  WorkspacePageLayout,
  WorkspacePageToolbar,
} from "@/components/WorkspacePageLayout";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { pushNotification } from "@/stores";
import {
  environmentNamePrefix,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import { useSQLReviewStore } from "@/stores/sqlReview";
import type { SQLReviewPolicy } from "@/types";
import { hasWorkspacePermissionV2, isDev, sqlReviewPolicySlug } from "@/utils";
import { sqlReviewV2Enabled } from "@/utils/featureGates";
import {
  SQLReviewAIReviewSection,
  useWorkspaceAIReview,
  useWorkspaceReviewAIPolicy,
} from "./SQLReviewAIReviewSection";
import {
  SQLReviewStandardRulesSection,
  useWorkspaceStandardRules,
} from "./SQLReviewStandardRulesSection";
import {
  SQLReviewVersionBadge,
  SQLReviewVersionSwitch,
} from "./SQLReviewVersionSwitch";

// The resources a policy is attached to, by kind, on one line: the
// environment and project names carry their own labels.
function PolicyResources({ resources }: { resources: string[] }) {
  const { t } = useTranslation();
  const groups = [
    {
      key: "environments",
      label: t("common.environments"),
      resources: resources.filter((resource) =>
        resource.startsWith(environmentNamePrefix)
      ),
    },
    {
      key: "projects",
      label: t("common.projects"),
      resources: resources.filter((resource) =>
        resource.startsWith(projectNamePrefix)
      ),
    },
  ].filter((group) => group.resources.length > 0);
  if (groups.length === 0) {
    return <span className="text-control-light">-</span>;
  }
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
      {groups.map((group) => (
        <span className="inline-flex items-center gap-x-2" key={group.key}>
          <span className="text-xs text-control-light">{group.label}</span>
          {group.resources.map((resource) => (
            <ResourceLink
              key={resource}
              resource={resource}
              showResourceType={false}
            />
          ))}
        </span>
      ))}
    </div>
  );
}

// ============================================================
// PolicyTable
// ============================================================

function PolicyTable({
  policies,
  searchText,
  onDelete,
}: {
  policies: SQLReviewPolicy[];
  searchText: string;
  onDelete: (policy: SQLReviewPolicy) => Promise<void>;
}) {
  const { t } = useTranslation();
  const sqlReviewStore = useSQLReviewStore();

  const hasUpdatePermission = hasWorkspacePermissionV2(
    "bb.reviewConfigs.update"
  );
  const hasDeletePermission = hasWorkspacePermissionV2(
    "bb.reviewConfigs.delete"
  );

  const navigateToDetail = useCallback((policy: SQLReviewPolicy) => {
    router.push({
      name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
      params: { sqlReviewPolicySlug: sqlReviewPolicySlug(policy) },
    });
  }, []);

  const toggleEnabled = useCallback(
    async (policy: SQLReviewPolicy, enabled: boolean) => {
      await sqlReviewStore.upsertReviewPolicy({
        id: policy.id,
        enforce: enabled,
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    },
    [sqlReviewStore, t]
  );

  // The policy whose deletion awaits confirmation; a failed delete keeps the
  // dialog open.
  const [pendingDelete, setPendingDelete] = useState<SQLReviewPolicy>();
  const [deleting, setDeleting] = useState(false);
  const confirmDelete = async () => {
    if (!pendingDelete) return;
    setDeleting(true);
    try {
      await onDelete(pendingDelete);
      setPendingDelete(undefined);
    } finally {
      setDeleting(false);
    }
  };

  return (
    <>
      {/* Desktop table */}
      <div className="hidden lg:block border rounded-sm overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-control-bg">
              <TableHead className="whitespace-nowrap">
                {t("common.name")}
              </TableHead>
              <TableHead className="whitespace-nowrap">
                {t("common.resource")}
              </TableHead>
              <TableHead className="whitespace-nowrap text-right">
                {t("sql-review.enabled-rules")}
              </TableHead>
              <TableHead className="whitespace-nowrap">
                {t("common.enabled")}
              </TableHead>
              <TableHead className="whitespace-nowrap text-right">
                {t("common.operations")}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {policies.map((policy) => (
              <TableRow key={policy.id}>
                <TableCell>
                  <HighlightLabelText text={policy.name} keyword={searchText} />
                </TableCell>
                <TableCell>
                  <PolicyResources resources={policy.resources} />
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {policy.ruleList.length}
                </TableCell>
                <TableCell>
                  {hasUpdatePermission ? (
                    <Switch
                      checked={policy.enforce}
                      onCheckedChange={(checked) =>
                        toggleEnabled(policy, checked)
                      }
                    />
                  ) : policy.enforce ? (
                    <Check className="w-4 h-4 text-control-light" />
                  ) : (
                    <X className="w-4 h-4 text-control-light" />
                  )}
                </TableCell>
                <TableCell>
                  <div className="flex items-center justify-end gap-x-2">
                    <Tooltip
                      content={
                        hasUpdatePermission
                          ? t("common.edit")
                          : t("common.view")
                      }
                    >
                      <Button
                        appearance="secondary"
                        aria-label={
                          hasUpdatePermission
                            ? t("common.edit")
                            : t("common.view")
                        }
                        size="sm"
                        onClick={() => navigateToDetail(policy)}
                      >
                        <Pencil className="size-4" />
                      </Button>
                    </Tooltip>
                    {hasDeletePermission && (
                      <Tooltip content={t("common.delete")}>
                        <Button
                          appearance="secondary"
                          aria-label={t("common.delete")}
                          size="sm"
                          className="text-control-light hover:text-error"
                          onClick={() => setPendingDelete(policy)}
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      </Tooltip>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {/* Mobile card layout */}
      <div className="flex flex-col lg:hidden border px-2 divide-y divide-block-border">
        {policies.map((policy) => (
          <div key={policy.id} className="py-4">
            <div className="text-md">{policy.name}</div>
            <div className="flex flex-wrap items-center mt-2 gap-2">
              <PolicyResources resources={policy.resources} />
              {!policy.enforce && (
                <Badge variant="warning">{t("common.disable")}</Badge>
              )}
            </div>
            <div className="flex items-center gap-x-2 mt-4">
              <Button
                appearance="outline"
                size="sm"
                onClick={() => navigateToDetail(policy)}
              >
                {hasUpdatePermission ? t("common.edit") : t("common.view")}
              </Button>
              {hasDeletePermission && (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setPendingDelete(policy)}
                >
                  {t("common.delete")}
                </Button>
              )}
            </div>
          </div>
        ))}
      </div>

      <AlertDialog
        open={pendingDelete !== undefined}
        onOpenChange={(open) => {
          if (!open && !deleting) setPendingDelete(undefined);
        }}
      >
        <AlertDialogContent className="max-w-lg">
          <AlertDialogTitle>
            {t("sql-review.delete-policy-title", { name: pendingDelete?.name })}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("sql-review.delete-description")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button
              appearance="outline"
              disabled={deleting}
              onClick={() => setPendingDelete(undefined)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              disabled={deleting}
              onClick={() => void confirmDelete()}
            >
              {t("common.delete")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

// ============================================================
// SQLReviewPage (exported)
// ============================================================

export function SQLReviewPage() {
  const { t } = useTranslation();
  const sqlReviewStore = useSQLReviewStore();
  const [searchText, setSearchText] = useState("");
  // With SQL Review V2 the standard rules replace the v1 review policies.
  const sqlReviewV2 = sqlReviewV2Enabled();
  const standardRules = useWorkspaceStandardRules(sqlReviewV2);
  const aiReview = useWorkspaceAIReview();
  const aiReviewPolicy = useWorkspaceReviewAIPolicy();
  const aiReviewPolicyVisible = aiReview.aiEnabled && aiReview.enabled;
  const formParts = [standardRules, aiReview, aiReviewPolicy];
  const saveParts = [
    standardRules,
    aiReview,
    ...(aiReviewPolicyVisible ? [aiReviewPolicy] : []),
  ];
  const isDirty = formParts.some((part) => part.isDirty);
  const saving = formParts.some((part) => part.saving);
  const hasSaveableChanges = saveParts.some((part) => part.isDirty);
  useUnsavedChangesGuard(isDirty);

  useEffect(() => {
    if (sqlReviewV2) return;
    useSQLReviewStore.getState().fetchReviewPolicyList();
  }, [sqlReviewV2]);

  const policyList = sqlReviewStore.reviewPolicyList;

  const filteredList = useMemo(() => {
    if (!searchText) return policyList;
    const lower = searchText.toLowerCase();
    return policyList.filter((p) => p.name.toLowerCase().includes(lower));
  }, [policyList, searchText]);

  const hasCreatePermission = hasWorkspacePermissionV2(
    "bb.reviewConfigs.create"
  );

  const navigateToCreate = useCallback(() => {
    router.push({ name: WORKSPACE_ROUTE_SQL_REVIEW_CREATE });
  }, []);

  const handleDelete = useCallback(
    async (policy: SQLReviewPolicy) => {
      await sqlReviewStore.removeReviewPolicy(policy.id);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-review.policy-removed"),
      });
    },
    [sqlReviewStore, t]
  );
  const versionBadge = isDev() ? <SQLReviewVersionBadge /> : undefined;
  const versionAction = isDev() ? <SQLReviewVersionSwitch /> : undefined;

  return (
    // The page fills the height so the footer sits at the bottom even when the
    // content is short.
    <WorkspacePageLayout className="min-h-full gap-y-4">
      {sqlReviewV2 ? (
        <>
          <SQLReviewStandardRulesSection
            standardRules={standardRules}
            titleSuffix={versionBadge}
            action={versionAction}
          />
          <SQLReviewAIReviewSection
            aiReview={aiReview}
            policy={aiReviewPolicy}
          />
        </>
      ) : (
        <div className="flex w-full flex-col gap-y-8">
          <FormSection
            description={
              <>
                {t("sql-review.description")}{" "}
                <LearnMoreLink href="https://docs.bytebase.com/sql-review/review-rules?source=console" />
              </>
            }
            layout="single-column"
            padding="none"
            title={
              versionBadge ? (
                <div className="flex flex-wrap items-center gap-3">
                  <span>{t("sql-review.policies")}</span> {versionBadge}
                </div>
              ) : (
                t("sql-review.policies")
              )
            }
            action={versionAction}
          >
            <div className="flex flex-col gap-y-4">
              <WorkspacePageToolbar>
                <SearchInput
                  value={searchText}
                  onChange={(e) => setSearchText(e.target.value)}
                  placeholder={t("common.filter-by-name")}
                  wrapperClassName="max-w-80"
                />
                {hasCreatePermission && (
                  <Button onClick={navigateToCreate}>
                    <Plus className="size-4" />
                    {t("common.create")}
                  </Button>
                )}
              </WorkspacePageToolbar>
              {policyList.length > 0 ? (
                <PolicyTable
                  policies={filteredList}
                  searchText={searchText}
                  onDelete={handleDelete}
                />
              ) : (
                <div className="py-12 border rounded-sm flex flex-col items-center justify-center gap-y-3 text-control-light">
                  <span>{t("common.no-data")}</span>
                </div>
              )}
            </div>
          </FormSection>

          <SQLReviewAIReviewSection
            aiReview={aiReview}
            policy={aiReviewPolicy}
            standardRules={false}
          />
        </div>
      )}

      {isDirty && (
        <StickyActionFooter
          className="-mb-4 mt-auto"
          contentPadding={false}
          left={
            <Button
              appearance="outline"
              disabled={saving}
              onClick={() => {
                for (const part of formParts) part.revert();
              }}
            >
              {t("common.cancel")}
            </Button>
          }
          right={
            <Button
              disabled={
                saving ||
                !hasSaveableChanges ||
                (aiReviewPolicyVisible && aiReviewPolicy.error !== undefined)
              }
              onClick={async () => {
                const saves = saveParts
                  .filter((part) => part.isDirty)
                  .map((part) => part.save());
                // One click reports one update, and only when every changed
                // part saved.
                if ((await Promise.all(saves)).every(Boolean)) {
                  pushNotification({
                    module: "bytebase",
                    style: "SUCCESS",
                    title: t("common.updated"),
                  });
                }
              }}
            >
              {t("common.update")}
            </Button>
          }
        />
      )}
    </WorkspacePageLayout>
  );
}
