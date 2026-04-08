import { X } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getRuleKey } from "@/components/SQLReview/components/utils";
import { Alert, AlertDescription } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import {
  pushNotification,
  useEnvironmentV1Store,
  useProjectV1Store,
  useSQLReviewStore,
} from "@/store";
import {
  environmentNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type { SQLReviewPolicy } from "@/types";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import type { RuleTemplateV2 } from "@/types/sqlReview";
import { isBuiltinRule, ruleTemplateMapV2 } from "@/types/sqlReview";
import { hasWorkspacePermissionV2 } from "@/utils";
import { RuleTableWithFilter } from "./RuleTable";
import { TabsByEngine } from "./TabsByEngine";

// ---------------------------------------------------------------------------
// RulesSelectPanel
// ---------------------------------------------------------------------------

interface RulesSelectPanelProps {
  show: boolean;
  selectedRuleMap: Map<Engine, Map<SQLReviewRule_Type, RuleTemplateV2>>;
  onClose: () => void;
  onRuleSelect: (rule: RuleTemplateV2) => void;
  onRuleRemove: (rule: RuleTemplateV2) => void;
}

export function RulesSelectPanel({
  show,
  selectedRuleMap,
  onClose,
  onRuleSelect,
  onRuleRemove,
}: RulesSelectPanelProps) {
  const { t } = useTranslation();
  useEscapeKey(show, onClose);

  const getSelectedRuleKeys = useCallback(
    (engine: Engine): string[] => {
      const map = selectedRuleMap.get(engine);
      if (!map) return [];
      const keys: string[] = [];
      for (const rule of map.values()) {
        keys.push(getRuleKey(rule));
      }
      return keys;
    },
    [selectedRuleMap]
  );

  const onSelectedRuleKeysUpdate = useCallback(
    (engine: Engine, keys: string[]) => {
      const oldKeys = new Set(getSelectedRuleKeys(engine));
      const newKeys = new Set(keys);

      for (const key of newKeys) {
        if (oldKeys.has(key)) {
          oldKeys.delete(key);
          continue;
        }
        const [engineStr, ruleKey] = key.split(":");
        const engineNum = parseInt(engineStr) as Engine;
        const ruleType = parseInt(ruleKey) as SQLReviewRule_Type;
        const rule = ruleTemplateMapV2.get(engineNum)?.get(ruleType);
        if (rule) {
          onRuleSelect(rule);
        }
      }

      // Remaining old keys were deselected
      for (const oldKey of oldKeys) {
        const [engineStr, ruleKey] = oldKey.split(":");
        const engineNum = parseInt(engineStr) as Engine;
        const ruleType = parseInt(ruleKey) as SQLReviewRule_Type;
        const template = ruleTemplateMapV2.get(engineNum)?.get(ruleType);
        if (template && isBuiltinRule(template)) {
          continue;
        }
        const rule = selectedRuleMap.get(engineNum)?.get(ruleType);
        if (rule) {
          onRuleRemove(rule);
        }
      }
    },
    [getSelectedRuleKeys, selectedRuleMap, onRuleSelect, onRuleRemove]
  );

  if (!show) return null;

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="ml-auto relative bg-white w-[70rem] max-w-[100vw] h-full shadow-lg flex flex-col">
        <div className="px-4 py-3 border-b flex items-center justify-between">
          <h2 className="text-lg font-medium">
            {t("sql-review.select-review-rules")}
          </h2>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="w-4 h-4" />
          </Button>
        </div>
        <div className="flex-1 overflow-y-auto p-4">
          <TabsByEngine ruleMapByEngine={ruleTemplateMapV2}>
            {(ruleList, engine) => (
              <RuleTableWithFilter
                engine={engine}
                ruleList={ruleList}
                editable={false}
                hideLevel
                supportSelect
                size="small"
                selectedRuleKeys={getSelectedRuleKeys(engine)}
                onSelectedRuleKeysChange={(keys) =>
                  onSelectedRuleKeysUpdate(engine, keys)
                }
              />
            )}
          </TabsByEngine>
        </div>
        <div className="px-4 py-3 border-t flex justify-end gap-x-3">
          <Button variant="outline" onClick={onClose}>
            {t("common.close")}
          </Button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// AttachResourcesPanel
// ---------------------------------------------------------------------------

interface AttachResourcesPanelProps {
  show: boolean;
  review: SQLReviewPolicy;
  onClose: () => void;
}

export function AttachResourcesPanel({
  show,
  review,
  onClose,
}: AttachResourcesPanelProps) {
  const { t } = useTranslation();
  const sqlReviewStore = useSQLReviewStore();
  const envStore = useEnvironmentV1Store();
  const projStore = useProjectV1Store();

  const environments = useVueState(() => envStore.environmentList ?? []);
  const projects = useVueState(() => [...(projStore.getProjectList() ?? [])]);

  const [resources, setResources] = useState<string[]>([]);

  // Fetch projects on mount so the checkbox list is populated
  useEffect(() => {
    projStore.fetchProjectList({});
  }, [projStore]);

  useEffect(() => {
    setResources([...review.resources]);
  }, [review.resources]);

  useEscapeKey(show, onClose);

  const hasPermission = hasWorkspacePermissionV2("bb.policies.update");

  const environmentNames = useMemo(
    () => resources.filter((r) => r.startsWith(environmentNamePrefix)),
    [resources]
  );

  const projectNames = useMemo(
    () => resources.filter((r) => r.startsWith(projectNamePrefix)),
    [resources]
  );

  const toggleResource = (name: string) => {
    setResources((prev) =>
      prev.includes(name) ? prev.filter((r) => r !== name) : [...prev, name]
    );
  };

  const getResourceAttachedConfigName = (resource: string): string | null => {
    const config = sqlReviewStore.getReviewPolicyByResouce(resource);
    if (config && config.id !== review.id) {
      return config.name;
    }
    return null;
  };

  // Track resources that are bound to other policies (for cache cleanup)
  const resourcesBindingWithOtherPolicy = useMemo(() => {
    const map = new Map<string, string[]>();
    for (const resource of resources) {
      const config = sqlReviewStore.getReviewPolicyByResouce(resource);
      if (config && config.id !== review.id) {
        if (!map.has(config.id)) {
          map.set(config.id, []);
        }
        map.get(config.id)!.push(resource);
      }
    }
    return map;
  }, [resources, sqlReviewStore, review.id]);

  const conflictingResources = useMemo(() => {
    return [...resourcesBindingWithOtherPolicy.values()].flat();
  }, [resourcesBindingWithOtherPolicy]);

  const [showOverrideConfirm, setShowOverrideConfirm] = useState(false);

  const onConfirm = () => {
    if (conflictingResources.length > 0) {
      setShowOverrideConfirm(true);
    } else {
      doSave();
    }
  };

  const doSave = async () => {
    const conflictMap = new Map(resourcesBindingWithOtherPolicy);
    await sqlReviewStore.upsertReviewConfigTag({
      oldResources: review.resources,
      newResources: resources,
      review: review.id,
    });
    if (conflictMap.size > 0) {
      sqlReviewStore.removeResourceForReview(conflictMap);
    }
    setShowOverrideConfirm(false);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("sql-review.policy-updated"),
    });
    onClose();
  };

  if (!show) return null;

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className="ml-auto relative bg-white w-[40rem] max-w-[100vw] h-full shadow-lg flex flex-col">
        <div className="px-4 py-3 border-b flex items-center justify-between">
          <h2 className="text-lg font-medium">
            {t("sql-review.attach-resource.self")}
          </h2>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="w-4 h-4" />
          </Button>
        </div>

        <div className="flex-1 overflow-y-auto p-4">
          <div className="flex flex-col gap-y-6">
            <p className="textinfolabel">
              {t("sql-review.attach-resource.label")}
            </p>

            {/* Environment section */}
            <div>
              <div className="textlabel mb-1">{t("common.environment")}</div>
              <p className="textinfolabel">
                {t("sql-review.attach-resource.label-environment")}
              </p>
              <div className="mt-3 flex flex-col gap-y-2">
                {environments.map((env) => {
                  const name = env.name;
                  const checked = environmentNames.includes(name);
                  const attachedConfig = getResourceAttachedConfigName(name);
                  return (
                    <label
                      key={name}
                      className="flex items-center gap-x-2 cursor-pointer"
                    >
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() => toggleResource(name)}
                        className="accent-accent"
                      />
                      <span>{env.title || name}</span>
                      {attachedConfig && (
                        <span className="opacity-60 textinfolabel">
                          ({attachedConfig})
                        </span>
                      )}
                    </label>
                  );
                })}
              </div>
            </div>

            {/* OR divider */}
            <div className="flex items-center gap-x-2">
              <div className="textlabel w-10 capitalize">{t("common.or")}</div>
              <hr className="flex-1 border-gray-200" />
            </div>

            {/* Project section */}
            <div>
              <div className="textlabel mb-1">{t("common.project")}</div>
              <p className="textinfolabel">
                {t("sql-review.attach-resource.label-project")}
              </p>
              <div className="mt-3 flex flex-col gap-y-2 max-h-60 overflow-y-auto">
                {projects.map((proj) => {
                  const name = proj.name;
                  const checked = projectNames.includes(name);
                  const attachedConfig = getResourceAttachedConfigName(name);
                  return (
                    <label
                      key={name}
                      className="flex items-center gap-x-2 cursor-pointer"
                    >
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() => toggleResource(name)}
                        className="accent-accent"
                      />
                      <span>{proj.title || name}</span>
                      {attachedConfig && (
                        <span className="opacity-60 textinfolabel">
                          ({attachedConfig})
                        </span>
                      )}
                    </label>
                  );
                })}
              </div>
            </div>
          </div>
        </div>

        {showOverrideConfirm && conflictingResources.length > 0 && (
          <div className="px-4 py-3 border-t">
            <Alert variant="warning">
              <AlertDescription>
                <p className="mb-2">
                  {t("sql-review.attach-resource.override-warning", {
                    button: t("common.confirm"),
                  })}
                </p>
                <ul className="list-disc list-inside text-sm">
                  {conflictingResources.map((r) => (
                    <li key={r}>{r}</li>
                  ))}
                </ul>
              </AlertDescription>
            </Alert>
          </div>
        )}

        <div className="px-4 py-3 border-t flex justify-end gap-x-2">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          {showOverrideConfirm ? (
            <Button variant="destructive" onClick={doSave}>
              {t("common.confirm")}
            </Button>
          ) : (
            <Button disabled={!hasPermission} onClick={onConfirm}>
              {t("common.confirm")}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
