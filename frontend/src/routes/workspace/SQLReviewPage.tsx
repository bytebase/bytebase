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
import { SQLReviewVersionSwitch } from "./SQLReviewVersionSwitch";

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
  onDelete: (policy: SQLReviewPolicy) => void;
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

  const [confirmingDelete, setConfirmingDelete] = useState<string | null>(null);

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
                    {hasDeletePermission &&
                      (confirmingDelete === policy.id ? (
                        <div className="flex items-center gap-x-2 text-xs">
                          <span>
                            {t("common.delete")} &apos;{policy.name}&apos;?
                          </span>
                          <Button
                            appearance="outline"
                            size="sm"
                            onClick={() => setConfirmingDelete(null)}
                          >
                            {t("common.cancel")}
                          </Button>
                          <Button
                            variant="destructive"
                            size="sm"
                            onClick={() => {
                              onDelete(policy);
                              setConfirmingDelete(null);
                            }}
                          >
                            {t("common.delete")}
                          </Button>
                        </div>
                      ) : (
                        <Tooltip content={t("common.delete")}>
                          <Button
                            appearance="secondary"
                            aria-label={t("common.delete")}
                            size="sm"
                            className="text-control-light hover:text-error"
                            onClick={() => setConfirmingDelete(policy.id)}
                          >
                            <Trash2 className="size-4" />
                          </Button>
                        </Tooltip>
                      ))}
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
                  onClick={() => {
                    if (
                      window.confirm(`${t("common.delete")} '${policy.name}'?`)
                    ) {
                      onDelete(policy);
                    }
                  }}
                >
                  {t("common.delete")}
                </Button>
              )}
            </div>
          </div>
        ))}
      </div>
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

  return (
    <WorkspacePageLayout className="gap-y-4">
      {isDev() && <SQLReviewVersionSwitch />}
      {sqlReviewV2 ? (
        <>
          <SQLReviewStandardRulesSection standardRules={standardRules} />
          <SQLReviewAIReviewSection
            aiReview={aiReview}
            policy={aiReviewPolicy}
          />
        </>
      ) : (
        // One content width for both sections, centered: on a wide screen the
        // short policy list and the AI review form share both edges instead of
        // spreading across it.
        <div className="mx-auto flex w-full max-w-6xl flex-col gap-y-8">
          <FormSection
            description={
              <>
                {t("sql-review.description")}{" "}
                <LearnMoreLink href="https://docs.bytebase.com/sql-review/review-rules?source=console" />
              </>
            }
            layout="single-column"
            padding="none"
            title={t("sql-review.policies")}
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
