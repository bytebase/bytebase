import { Trash2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { rulesToTemplate } from "@/components/SQLReview/components/utils";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { AttachResourcesPanel } from "@/react/components/sql-review/Panels";
import { ResourceLink } from "@/react/components/sql-review/ResourceLink";
import { ReviewCreation } from "@/react/components/sql-review/ReviewCreation";
import { RuleTableWithFilter } from "@/react/components/sql-review/RuleTable";
import { TabsByEngine } from "@/react/components/sql-review/TabsByEngine";
import { Alert } from "@/react/components/ui/alert";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { WORKSPACE_ROUTE_SQL_REVIEW } from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useSQLReviewStore } from "@/store";
import type { RuleTemplateV2 } from "@/types";
import {
  convertRuleMapToPolicyRuleList,
  getRuleMapByEngine,
  isBuiltinRule,
  UNKNOWN_ID,
  withBuiltinRules,
} from "@/types";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import {
  hasWorkspacePermissionV2,
  setDocumentTitle,
  sqlReviewNameFromSlug,
} from "@/utils";

// ============================================================
// SQLReviewDetailPage
// ============================================================

export function SQLReviewDetailPage({
  sqlReviewPolicySlug,
}: {
  sqlReviewPolicySlug: string;
}) {
  const { t } = useTranslation();
  const store = useSQLReviewStore();

  const sqlReviewName = useMemo(
    () => sqlReviewNameFromSlug(sqlReviewPolicySlug),
    [sqlReviewPolicySlug]
  );

  // Fetch policy
  useEffect(() => {
    store.getOrFetchReviewPolicyByName(sqlReviewName, false);
  }, [store, sqlReviewName]);

  const reviewPolicy = useVueState(
    () =>
      store.getReviewPolicyByName(sqlReviewName) ?? {
        id: `${UNKNOWN_ID}`,
        enforce: false,
        name: "",
        ruleList: [],
        resources: [],
      }
  );

  // Set document title
  useEffect(() => {
    if (reviewPolicy.name) {
      setDocumentTitle(reviewPolicy.name);
    }
  }, [reviewPolicy.name]);

  const ruleListOfPolicy = useMemo((): RuleTemplateV2[] => {
    if (reviewPolicy.id === `${UNKNOWN_ID}`) return [];
    return rulesToTemplate(reviewPolicy).ruleList;
  }, [reviewPolicy]);

  // State
  const [editMode, setEditMode] = useState(false);
  const [editingTitle, setEditingTitle] = useState(false);
  const [showDisableModal, setShowDisableModal] = useState(false);
  const [showEnableModal, setShowEnableModal] = useState(false);
  const [showResourcePanel, setShowResourcePanel] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [rulesUpdated, setRulesUpdated] = useState(false);
  const [ruleMapByEngine, setRuleMapByEngine] = useState<
    Map<Engine, Map<SQLReviewRule_Type, RuleTemplateV2>>
  >(new Map());

  // Sync rule map when policy changes
  useEffect(() => {
    setRuleMapByEngine(withBuiltinRules(getRuleMapByEngine(ruleListOfPolicy)));
    setRulesUpdated(false);
  }, [ruleListOfPolicy]);

  // Check if attachResourcePanel query param is set
  useEffect(() => {
    const url = new URL(window.location.href);
    if (
      url.searchParams.get("attachResourcePanel") &&
      hasWorkspacePermissionV2("bb.policies.update")
    ) {
      setShowResourcePanel(true);
    }
  }, []);

  const hasUpdatePermission = hasWorkspacePermissionV2(
    "bb.reviewConfigs.update"
  );
  const hasDeletePermission = hasWorkspacePermissionV2(
    "bb.reviewConfigs.delete"
  );

  const pushUpdatedNotify = useCallback(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("sql-review.policy-updated"),
    });
  }, [t]);

  const changeName = useCallback(
    async (title: string) => {
      setEditingTitle(false);
      if (title === reviewPolicy.name || !title.trim()) return;
      await store.upsertReviewPolicy({ id: reviewPolicy.id, title });
      pushUpdatedNotify();
    },
    [store, reviewPolicy.id, reviewPolicy.name, pushUpdatedNotify]
  );

  const markChange = useCallback(
    (rule: RuleTemplateV2, overrides: Partial<RuleTemplateV2>) => {
      setRuleMapByEngine((prev) => {
        const newMap = new Map(prev);
        const engineMap = new Map(newMap.get(rule.engine) || new Map());
        const existing = engineMap.get(rule.type);
        if (!existing) return prev;
        engineMap.set(rule.type, { ...existing, ...overrides });
        newMap.set(rule.engine, engineMap);
        return newMap;
      });
      setRulesUpdated(true);
    },
    []
  );

  const removeRule = useCallback((rule: RuleTemplateV2) => {
    if (isBuiltinRule(rule)) return;
    setRuleMapByEngine((prev) => {
      const newMap = new Map(prev);
      const engineMap = new Map(newMap.get(rule.engine) || new Map());
      engineMap.delete(rule.type);
      if (engineMap.size === 0) newMap.delete(rule.engine);
      else newMap.set(rule.engine, engineMap);
      return newMap;
    });
    setRulesUpdated(true);
  }, []);

  const onCancelChanges = useCallback(() => {
    setRuleMapByEngine(withBuiltinRules(getRuleMapByEngine(ruleListOfPolicy)));
    setRulesUpdated(false);
  }, [ruleListOfPolicy]);

  const onApplyChanges = useCallback(async () => {
    await store.upsertReviewPolicy({
      id: reviewPolicy.id,
      title: reviewPolicy.name,
      ruleList: convertRuleMapToPolicyRuleList(ruleMapByEngine),
    });
    setRulesUpdated(false);
    pushUpdatedNotify();
  }, [
    store,
    reviewPolicy.id,
    reviewPolicy.name,
    ruleMapByEngine,
    pushUpdatedNotify,
  ]);

  const onArchive = useCallback(async () => {
    await store.upsertReviewPolicy({ id: reviewPolicy.id, enforce: false });
    setShowDisableModal(false);
    pushUpdatedNotify();
  }, [store, reviewPolicy.id, pushUpdatedNotify]);

  const onRestore = useCallback(async () => {
    await store.upsertReviewPolicy({ id: reviewPolicy.id, enforce: true });
    setShowEnableModal(false);
    pushUpdatedNotify();
  }, [store, reviewPolicy.id, pushUpdatedNotify]);

  const onRemove = useCallback(async () => {
    await store.removeReviewPolicy(reviewPolicy.id);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("sql-review.policy-removed"),
    });
    router.replace({ name: WORKSPACE_ROUTE_SQL_REVIEW });
  }, [store, reviewPolicy.id, t]);

  // Edit mode: show the wizard
  if (editMode) {
    return (
      <div className="px-4 py-4">
        <ReviewCreation
          policy={reviewPolicy}
          name={reviewPolicy.name}
          selectedRuleList={ruleListOfPolicy}
          selectedResources={reviewPolicy.resources}
          onCancel={() => setEditMode(false)}
        />
      </div>
    );
  }

  return (
    <div className="px-4 py-4">
      {/* Disabled warning */}
      {!reviewPolicy.enforce && (
        <Alert
          variant="warning"
          className="mb-4"
          description={t("sql-review.disabled")}
        />
      )}

      {/* Header: title + actions */}
      <div className="flex flex-col gap-y-2 items-start md:items-center gap-x-2 justify-center md:flex-row">
        {editingTitle ? (
          <Input
            className="flex-1 text-xl font-bold"
            defaultValue={reviewPolicy.name}
            autoFocus
            onBlur={(e) => changeName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter")
                changeName((e.target as HTMLInputElement).value);
            }}
          />
        ) : (
          <h1
            className="flex-1 text-xl font-bold truncate cursor-text px-0.5"
            onClick={() => {
              if (hasUpdatePermission) {
                setEditingTitle(true);
              }
            }}
          >
            {reviewPolicy.name}
          </h1>
        )}
        <div className="flex gap-x-2">
          <PermissionGuard permissions={["bb.reviewConfigs.update"]}>
            {reviewPolicy.enforce ? (
              <Button
                variant="outline"
                disabled={!hasUpdatePermission}
                onClick={() => setShowDisableModal(true)}
              >
                {t("common.disable")}
              </Button>
            ) : (
              <Button
                variant="outline"
                disabled={!hasUpdatePermission}
                onClick={() => setShowEnableModal(true)}
              >
                {t("common.enable")}
              </Button>
            )}
          </PermissionGuard>
          <PermissionGuard permissions={["bb.policies.update"]}>
            <Button
              variant="outline"
              disabled={!hasWorkspacePermissionV2("bb.policies.update")}
              onClick={() => setShowResourcePanel(true)}
            >
              {t("sql-review.attach-resource.change-resources")}
            </Button>
          </PermissionGuard>
          <PermissionGuard permissions={["bb.reviewConfigs.update"]}>
            <Button
              disabled={!hasUpdatePermission}
              onClick={() => setEditMode(true)}
            >
              {t("sql-review.create.configure-rule.change-template")}
            </Button>
          </PermissionGuard>
        </div>
      </div>

      {/* Attached resources */}
      <div className="mt-4 flex flex-col gap-y-4">
        {reviewPolicy.resources.length === 0 && (
          <Alert
            variant="warning"
            description={
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-medium">
                    {t("sql-review.attach-resource.no-linked-resources")}
                  </p>
                  <p className="text-sm mt-1">
                    {t("sql-review.attach-resource.label")}
                  </p>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowResourcePanel(true)}
                >
                  {t("sql-review.attach-resource.self")}
                </Button>
              </div>
            }
          />
        )}
        <div className="flex flex-wrap gap-y-2 gap-x-2">
          {reviewPolicy.resources.map((resource) => (
            <Badge key={resource} variant="default">
              <ResourceLink resource={resource} />
            </Badge>
          ))}
        </div>
      </div>

      {/* Rules by engine */}
      <div className="mt-5">
        <TabsByEngine ruleMapByEngine={ruleMapByEngine}>
          {(ruleList, engine) => (
            <RuleTableWithFilter
              engine={engine}
              ruleList={ruleList}
              editable={hasUpdatePermission}
              onRuleUpsert={markChange}
              onRuleRemove={removeRule}
            />
          )}
        </TabsByEngine>
      </div>

      {/* Delete button */}
      <hr className="my-6" />
      <PermissionGuard permissions={["bb.reviewConfigs.delete"]}>
        <Button
          variant="destructive"
          disabled={!hasDeletePermission}
          onClick={() => setShowDeleteConfirm(true)}
        >
          <Trash2 className="w-4 h-4 mr-1" />
          {t("sql-review.delete")}
        </Button>
      </PermissionGuard>

      {/* Sticky save bar when rules changed */}
      {rulesUpdated && (
        <div className="w-full mt-4 py-2 border-t border-control-border flex justify-between bg-background sticky bottom-0 z-10">
          <Button variant="outline" onClick={onCancelChanges}>
            {t("common.cancel")}
          </Button>
          <Button onClick={onApplyChanges}>{t("common.update")}</Button>
        </div>
      )}

      {/* Disable confirmation dialog */}
      <Dialog open={showDisableModal} onOpenChange={setShowDisableModal}>
        <DialogContent className="max-w-md p-6">
          <DialogTitle>
            {t("common.disable")} &apos;{reviewPolicy.name}&apos;?
          </DialogTitle>
          <DialogDescription className="mt-2">
            {t("sql-review.disable-description")}
          </DialogDescription>
          <div className="flex justify-end gap-x-2 mt-6">
            <Button
              variant="outline"
              onClick={() => setShowDisableModal(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={onArchive}>
              {t("common.disable")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Enable confirmation dialog */}
      <Dialog open={showEnableModal} onOpenChange={setShowEnableModal}>
        <DialogContent className="max-w-md p-6">
          <DialogTitle>
            {t("common.enable")} &apos;{reviewPolicy.name}&apos;?
          </DialogTitle>
          <DialogDescription className="mt-2">
            {t("sql-review.enable-description")}
          </DialogDescription>
          <div className="flex justify-end gap-x-2 mt-6">
            <Button variant="outline" onClick={() => setShowEnableModal(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={onRestore}>{t("common.enable")}</Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Delete confirmation dialog */}
      <Dialog open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
        <DialogContent className="max-w-md p-6">
          <DialogTitle>
            {t("common.delete")} &apos;{reviewPolicy.name}&apos;?
          </DialogTitle>
          <DialogDescription className="mt-2">
            {t("sql-review.delete-description")}
          </DialogDescription>
          <div className="flex justify-end gap-x-2 mt-6">
            <Button
              variant="outline"
              onClick={() => setShowDeleteConfirm(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={onRemove}>
              {t("common.delete")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Attach resources panel */}
      <AttachResourcesPanel
        show={showResourcePanel}
        review={reviewPolicy}
        onClose={() => setShowResourcePanel(false)}
      />
    </div>
  );
}
