import {
  Check,
  ExternalLink,
  Pencil,
  Plus,
  Search,
  Trash2,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ResourceLink } from "@/react/components/sql-review/ResourceLink";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
  WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
} from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useSQLReviewStore } from "@/store";
import type { SQLReviewPolicy } from "@/types";
import { hasWorkspacePermissionV2, sqlReviewPolicySlug } from "@/utils";

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

  const highlight = useCallback(
    (text: string) => {
      if (!searchText) return text;
      const regex = new RegExp(
        `(${searchText.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`,
        "gi"
      );
      const parts = text.split(regex);
      return parts.map((part, i) =>
        regex.test(part) ? (
          <mark key={i} className="bg-yellow-100">
            {part}
          </mark>
        ) : (
          part
        )
      );
    },
    [searchText]
  );

  const [confirmingDelete, setConfirmingDelete] = useState<string | null>(null);

  return (
    <>
      {/* Desktop table */}
      <div className="hidden lg:block border rounded-sm overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b bg-control-bg">
              <th className="px-4 py-2 text-left font-medium whitespace-nowrap w-[200px]">
                {t("common.name")}
              </th>
              <th className="px-4 py-2 text-left font-medium whitespace-nowrap w-[250px]">
                {t("common.resource")}
              </th>
              <th className="px-4 py-2 text-left font-medium whitespace-nowrap">
                {t("sql-review.enabled-rules")}
              </th>
              <th className="px-4 py-2 text-left font-medium whitespace-nowrap capitalize w-[7rem]">
                {t("common.enabled")}
              </th>
              <th className="px-4 py-2 text-left font-medium whitespace-nowrap w-[10rem]">
                {t("common.operations")}
              </th>
            </tr>
          </thead>
          <tbody>
            {policies.map((policy, i) => (
              <tr
                key={policy.id}
                className={`border-b last:border-b-0 ${i % 2 === 1 ? "bg-gray-50" : ""}`}
              >
                <td className="px-4 py-2">{highlight(policy.name)}</td>
                <td className="px-4 py-2">
                  <div className="flex flex-wrap gap-2">
                    {policy.resources.length === 0 && <span>-</span>}
                    {policy.resources.map((resource) => (
                      <Badge key={resource} variant="default">
                        <ResourceLink resource={resource} />
                      </Badge>
                    ))}
                  </div>
                </td>
                <td className="px-4 py-2">{policy.ruleList.length}</td>
                <td className="px-4 py-2">
                  {hasUpdatePermission ? (
                    <input
                      type="checkbox"
                      checked={policy.enforce}
                      onChange={(e) => toggleEnabled(policy, e.target.checked)}
                      className="h-4 w-4 rounded-xs border-control-border accent-accent"
                    />
                  ) : policy.enforce ? (
                    <Check className="w-4 h-4 text-control-light" />
                  ) : (
                    <X className="w-4 h-4 text-control-light" />
                  )}
                </td>
                <td className="px-4 py-2">
                  <div className="flex items-center gap-x-2">
                    {hasDeletePermission &&
                      (confirmingDelete === policy.id ? (
                        <div className="flex items-center gap-x-1 text-xs">
                          <span>
                            {t("common.delete")} &apos;{policy.name}&apos;?
                          </span>
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
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setConfirmingDelete(null)}
                          >
                            {t("common.cancel")}
                          </Button>
                        </div>
                      ) : (
                        <Tooltip content={t("common.delete")}>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7 text-error hover:text-error"
                            onClick={() => setConfirmingDelete(policy.id)}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </Tooltip>
                      ))}
                    <Tooltip
                      content={
                        hasUpdatePermission
                          ? t("common.edit")
                          : t("common.view")
                      }
                    >
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7"
                        onClick={() => navigateToDetail(policy)}
                      >
                        <Pencil className="w-4 h-4" />
                      </Button>
                    </Tooltip>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Mobile card layout */}
      <div className="flex flex-col lg:hidden border px-2 divide-y divide-block-border">
        {policies.map((policy) => (
          <div key={policy.id} className="py-4">
            <div className="text-md">{policy.name}</div>
            <div className="flex flex-wrap mt-2 gap-2">
              {policy.resources.map((resource) => (
                <Badge key={resource} variant="default" className="text-xs">
                  <ResourceLink resource={resource} />
                </Badge>
              ))}
              {!policy.enforce && (
                <Badge variant="warning">{t("common.disable")}</Badge>
              )}
            </div>
            <div className="flex items-center gap-x-2 mt-4">
              <Button
                variant="outline"
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

  useEffect(() => {
    sqlReviewStore.fetchReviewPolicyList();
  }, [sqlReviewStore]);

  const policyList = useVueState(() => [...sqlReviewStore.reviewPolicyList]);

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
    <div className="px-4 py-4 mx-auto flex flex-col gap-y-4">
      <div className="textinfolabel">
        {t("sql-review.description")}{" "}
        <a
          href="https://docs.bytebase.com/sql-review/review-rules?source=console"
          target="_blank"
          rel="noopener noreferrer"
          className="normal-link inline-flex items-center gap-x-1"
        >
          {t("common.learn-more")}
          <ExternalLink className="w-4 h-4" />
        </a>
      </div>

      <div className="flex justify-end items-center gap-x-2">
        <div className="relative max-w-full">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-control-light pointer-events-none" />
          <Input
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            placeholder={t("common.filter-by-name")}
            className="pl-8"
            autoFocus
          />
        </div>
        {hasCreatePermission && (
          <Button onClick={navigateToCreate}>
            <Plus className="w-4 h-4 mr-1" />
            {t("sql-review.create-policy")}
          </Button>
        )}
      </div>

      {policyList.length > 0 ? (
        <PolicyTable
          policies={filteredList}
          searchText={searchText}
          onDelete={handleDelete}
        />
      ) : (
        <div className="py-12 border rounded-sm flex flex-col items-center justify-center gap-y-3 text-control-light">
          <span>{t("common.no-data")}</span>
          {hasCreatePermission && (
            <Button size="sm" onClick={navigateToCreate}>
              <Plus className="w-4 h-4 mr-1" />
              {t("sql-review.create-policy")}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
