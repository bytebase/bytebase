import { create } from "@bufbuild/protobuf";
import { isEmpty } from "lodash-es";
import { Pencil, Trash2, Undo2, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { ClassificationTree } from "@/react/components/ClassificationTree";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { LearnMoreLink } from "@/react/components/LearnMoreLink";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import classificationExample from "@/react/lib/sensitive-data/classification-example.json";
import {
  pushNotification,
  useSettingV1Store,
  useSubscriptionV1Store,
} from "@/store";
import type {
  DataClassificationSetting_DataClassificationConfig_DataClassification as DataClassification,
  DataClassificationSetting_DataClassificationConfig,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  DataClassificationSetting_DataClassificationConfig_DataClassificationSchema,
  DataClassificationSetting_DataClassificationConfig_LevelSchema,
  DataClassificationSetting_DataClassificationConfigSchema,
  DataClassificationSettingSchema,
  Setting_SettingName,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

interface ClassificationJSON {
  title: string;
  levels: { title: string; level: number }[];
  classification: {
    [key: string]: { id: string; title: string; level?: number };
  };
}

function configToJSON(
  config: DataClassificationSetting_DataClassificationConfig
): string {
  const data: ClassificationJSON = {
    title: config.title,
    levels: config.levels.map((l) => ({
      title: l.title,
      level: l.level,
    })),
    classification: {},
  };
  for (const [key, val] of Object.entries(config.classification)) {
    const entry: ClassificationJSON["classification"][string] = {
      id: val.id,
      title: val.title,
    };
    if (val.level !== undefined && val.level !== 0) {
      entry.level = val.level;
    }
    data.classification[key] = entry;
  }
  return JSON.stringify(data, null, 2);
}

// --- Monaco Editor wrapper (lazy-loaded) ---

function MonacoJSONEditor({
  value,
  readonly,
  onChange,
}: {
  value: string;
  readonly: boolean;
  onChange: (v: string) => void;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  // biome-ignore lint/suspicious/noExplicitAny: Monaco editor instance type
  const editorRef = useRef<any>(null); // eslint-disable-line @typescript-eslint/no-explicit-any
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  useEffect(() => {
    let disposed = false;
    (async () => {
      const { createMonacoEditor } = await import(
        "@/react/components/monaco/core"
      );
      if (disposed || !containerRef.current) return;
      const editor = await createMonacoEditor({
        container: containerRef.current,
        options: {
          language: "json",
          value,
          readOnly: readonly,
        },
      });
      if (disposed) {
        editor.dispose();
        return;
      }
      editorRef.current = editor;
      editor.onDidChangeModelContent(() => {
        onChangeRef.current(editor.getValue());
      });
    })();
    return () => {
      disposed = true;
      editorRef.current?.dispose();
      editorRef.current = null;
    };
    // Only run on mount — editor is created once and updated via separate effects
  }, []);

  // Sync value from outside (revert / cancel)
  useEffect(() => {
    const editor = editorRef.current;
    if (editor && editor.getValue() !== value) {
      editor.setValue(value);
    }
  }, [value]);

  // Sync readonly
  useEffect(() => {
    editorRef.current?.updateOptions({ readOnly: readonly });
  }, [readonly]);

  return <div ref={containerRef} className="w-full h-full" />;
}

// --- Main page ---

export function DataClassificationPage() {
  const { t } = useTranslation();
  const settingStore = useSettingV1Store();
  const subscriptionStore = useSubscriptionV1Store();

  const hasClassificationFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(
      PlanFeature.FEATURE_DATA_CLASSIFICATION
    )
  );
  const allowEdit = hasWorkspacePermissionV2("bb.settings.set");

  const classificationConfigs = useVueState(() => settingStore.classification);

  const [loaded, setLoaded] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editorContent, setEditorContent] = useState("");
  const [showClearConfirm, setShowClearConfirm] = useState(false);

  // The effective config from the store
  const configId = useRef(uuidv4());
  const formerConfig = useMemo(() => {
    const first = classificationConfigs?.[0];
    return create(DataClassificationSetting_DataClassificationConfigSchema, {
      id: first?.id || configId.current,
      title: first?.title || "",
      levels: first?.levels || [],
      classification: first?.classification || {},
    });
  }, [classificationConfigs]);

  const emptyConfig = Object.keys(formerConfig.classification).length === 0;

  const savedContent = useMemo(() => {
    if (emptyConfig) {
      return JSON.stringify(classificationExample, null, 2);
    }
    return configToJSON(formerConfig);
  }, [emptyConfig, formerConfig]);

  // Fetch setting on mount
  useEffect(() => {
    settingStore
      .getOrFetchSettingByName(Setting_SettingName.DATA_CLASSIFICATION)
      .then(() => setLoaded(true));
  }, [settingStore]);

  // Sync editor content when not editing
  useEffect(() => {
    if (!editing) {
      setEditorContent(savedContent);
    }
  }, [editing, savedContent]);

  const editorDirty = editorContent !== savedContent;

  const onRevert = () => setEditorContent(savedContent);
  const onCancelEdit = () => {
    setEditorContent(savedContent);
    setEditing(false);
  };

  const upsertSetting = useCallback(
    async (configs: DataClassificationSetting_DataClassificationConfig[]) => {
      await settingStore.upsertSetting({
        name: Setting_SettingName.DATA_CLASSIFICATION,
        value: create(SettingValueSchema, {
          value: {
            case: "dataClassification",
            value: create(DataClassificationSettingSchema, {
              configs,
            }),
          },
        }),
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    },
    [settingStore, t]
  );

  const onSave = async () => {
    let data: ClassificationJSON;
    try {
      data = JSON.parse(editorContent);
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("settings.sensitive-data.classification.invalid-json"),
      });
      return;
    }

    if (isEmpty(data.classification) || Array.isArray(data.classification)) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t(
          "settings.sensitive-data.classification.missing-classification"
        ),
      });
      return;
    }
    if (Object.keys(data.classification).length === 0) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("settings.sensitive-data.classification.empty-classification"),
      });
      return;
    }
    if (!Array.isArray(data.levels) || data.levels.length === 0) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("settings.sensitive-data.classification.missing-levels"),
      });
      return;
    }

    const config = create(
      DataClassificationSetting_DataClassificationConfigSchema,
      {
        id: formerConfig.id || uuidv4(),
        title: data.title || "",
        levels: data.levels.map((level) =>
          create(
            DataClassificationSetting_DataClassificationConfig_LevelSchema,
            level
          )
        ),
        classification: Object.values(data.classification).reduce(
          (map, item) => {
            map[item.id] = create(
              DataClassificationSetting_DataClassificationConfig_DataClassificationSchema,
              item
            );
            return map;
          },
          {} as { [key: string]: DataClassification }
        ),
      }
    );

    await upsertSetting([config]);
    setEditing(false);
  };

  const onClear = async () => {
    await upsertSetting([]);
    setShowClearConfirm(false);
  };

  if (!loaded) return null;

  return (
    <div className="w-full px-4 py-4 flex flex-col gap-y-4">
      {!hasClassificationFeature && (
        <FeatureAttention feature={PlanFeature.FEATURE_DATA_CLASSIFICATION} />
      )}

      <div className="flex items-center justify-between">
        <div className="text-sm text-control-light">
          {t("database.classification.description")}
          <LearnMoreLink
            href="https://docs.bytebase.com/security/data-masking/data-classification?source=console"
            className="ml-1 text-accent"
          />
        </div>

        {allowEdit && (
          <div className="flex items-center justify-end gap-x-2 shrink-0">
            {editing ? (
              <>
                <Button
                  variant="outline"
                  disabled={!editorDirty}
                  onClick={onRevert}
                >
                  <Undo2 className="w-4 h-4 mr-1" />
                  {t("common.revert")}
                </Button>
                <Button variant="outline" onClick={onCancelEdit}>
                  <X className="w-4 h-4 mr-1" />
                  {t("common.cancel")}
                </Button>
                <Button
                  disabled={
                    (!editorDirty && !emptyConfig) || !hasClassificationFeature
                  }
                  onClick={onSave}
                >
                  {t("common.save")}
                </Button>
              </>
            ) : (
              <>
                {!emptyConfig && (
                  <div className="relative">
                    {showClearConfirm ? (
                      <div className="flex items-center gap-x-2">
                        <span className="text-sm text-control">
                          {t("bbkit.confirm-button.sure-to-delete")}
                        </span>
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={onClear}
                        >
                          {t("common.clear")}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setShowClearConfirm(false)}
                        >
                          {t("common.cancel")}
                        </Button>
                      </div>
                    ) : (
                      <Button
                        variant="outline"
                        disabled={!hasClassificationFeature}
                        onClick={() => setShowClearConfirm(true)}
                      >
                        <Trash2 className="w-4 h-4 mr-1" />
                        {t("common.clear")}
                      </Button>
                    )}
                  </div>
                )}
                <Button
                  disabled={!hasClassificationFeature}
                  onClick={() => setEditing(true)}
                >
                  <Pencil className="w-4 h-4 mr-1" />
                  {t("common.edit")}
                </Button>
              </>
            )}
          </div>
        )}
      </div>

      {editing && (
        <>
          <div className="text-sm text-control-light flex flex-col gap-1">
            <p>{t("settings.sensitive-data.classification.guide-intro")}</p>
            <ul className="list-disc list-inside">
              <li>
                {t("settings.sensitive-data.classification.guide-levels")}
              </li>
              <li>
                {t(
                  "settings.sensitive-data.classification.guide-classification"
                )}
              </li>
              <li>
                {t("settings.sensitive-data.classification.guide-hierarchy")}
              </li>
            </ul>
          </div>
          <div
            className="border rounded-sm overflow-hidden"
            style={{ height: "50vh" }}
          >
            <MonacoJSONEditor
              value={editorContent}
              readonly={!allowEdit || !hasClassificationFeature}
              onChange={setEditorContent}
            />
          </div>
        </>
      )}

      {!editing && !emptyConfig && <ClassificationTree config={formerConfig} />}
    </div>
  );
}
