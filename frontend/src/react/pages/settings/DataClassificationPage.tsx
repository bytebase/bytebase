import { create } from "@bufbuild/protobuf";
import { isEmpty } from "lodash-es";
import {
  ChevronRight,
  ExternalLink,
  Pencil,
  Trash2,
  Undo2,
  X,
} from "lucide-react";
import {
  type JSX,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import classificationExample from "@/components/SensitiveData/classification-example.json";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { Button } from "@/react/components/ui/button";
import { SearchInput } from "@/react/components/ui/search-input";
import { useVueState } from "@/react/hooks/useVueState";
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
        "@/components/MonacoEditor/editor"
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

// --- Classification tree ---

interface TreeNode {
  key: string;
  label: string;
  level?: number;
  children: TreeNode[];
}

function sortClassification(a: { id: string }, b: { id: string }): number {
  const id1s = a.id.split("-");
  const id2s = b.id.split("-");
  if (id1s.length !== id2s.length) return id1s.length - id2s.length;
  for (let i = 0; i < id1s.length; i++) {
    if (id1s[i] === id2s[i]) continue;
    if (Number.isNaN(Number(id1s[i])) || Number.isNaN(Number(id2s[i]))) {
      return id1s[i].localeCompare(id2s[i]);
    }
    return Number(id1s[i]) - Number(id2s[i]);
  }
  return 0;
}

interface ClassificationMap {
  [key: string]: {
    id: string;
    label: string;
    level?: number;
    children: ClassificationMap;
  };
}

function buildTreeData(
  config: DataClassificationSetting_DataClassificationConfig
): TreeNode[] {
  const classifications = Object.values(config.classification).sort(
    sortClassification
  );
  const map: ClassificationMap = {};
  for (const c of classifications) {
    const ids = c.id.split("-");
    let tmp = map;
    for (let i = 0; i < ids.length - 1; i++) {
      const parentKey = ids.slice(0, i + 1).join("-");
      if (!tmp[parentKey]) break;
      tmp = tmp[parentKey].children;
    }
    tmp[c.id] = {
      id: c.id,
      label: c.title,
      level: c.level,
      children: {},
    };
  }

  function toNodes(m: ClassificationMap): TreeNode[] {
    return Object.values(m)
      .sort(sortClassification)
      .map((item) => ({
        key: item.id,
        label: `${item.id} ${item.label}`,
        level: item.level,
        children: toNodes(item.children),
      }));
  }
  return toNodes(map);
}

const bgColorList = [
  "bg-green-200",
  "bg-yellow-200",
  "bg-orange-300",
  "bg-amber-500",
  "bg-red-500",
];

function LevelBadge({
  level,
  config,
}: {
  level: number;
  config: DataClassificationSetting_DataClassificationConfig;
}) {
  const levelObj = config.levels.find((l) => l.level === level);
  if (!levelObj) return null;
  const color = bgColorList[level - 1] ?? "bg-gray-200";
  return (
    <span className={`ml-1 px-1 py-0.5 rounded-xs text-xs ${color}`}>
      {levelObj.title}
    </span>
  );
}

function highlightText(text: string, keyword: string) {
  if (!keyword) return text;
  const lower = text.toLowerCase();
  const kw = keyword.toLowerCase();
  const parts: (string | JSX.Element)[] = [];
  let cursor = 0;
  let idx = lower.indexOf(kw, cursor);
  while (idx !== -1) {
    if (idx > cursor) parts.push(text.slice(cursor, idx));
    parts.push(
      <b key={idx} className="text-accent">
        {text.slice(idx, idx + keyword.length)}
      </b>
    );
    cursor = idx + keyword.length;
    idx = lower.indexOf(kw, cursor);
  }
  if (cursor < text.length) parts.push(text.slice(cursor));
  return <>{parts}</>;
}

function matchesSearch(node: TreeNode, keyword: string): boolean {
  if (!keyword) return true;
  const lower = keyword.toLowerCase();
  if (node.label.toLowerCase().includes(lower)) return true;
  return node.children.some((c) => matchesSearch(c, lower));
}

function ClassificationTreeNode({
  node,
  config,
  searchText,
  defaultExpanded,
}: {
  node: TreeNode;
  config: DataClassificationSetting_DataClassificationConfig;
  searchText: string;
  defaultExpanded: boolean;
}) {
  const [expanded, setExpanded] = useState(defaultExpanded);
  const hasChildren = node.children.length > 0;

  // Auto-expand when searching
  useEffect(() => {
    if (searchText) setExpanded(true);
    else setExpanded(defaultExpanded);
  }, [searchText, defaultExpanded]);

  const filteredChildren = useMemo(
    () => node.children.filter((c) => matchesSearch(c, searchText)),
    [node.children, searchText]
  );

  return (
    <div>
      <div
        className="flex items-center gap-1 py-1 px-1 rounded-xs hover:bg-control-bg cursor-pointer select-none"
        onClick={() => hasChildren && setExpanded(!expanded)}
      >
        {hasChildren ? (
          <ChevronRight
            className={`w-4 h-4 shrink-0 transition-transform ${expanded ? "rotate-90" : ""}`}
          />
        ) : (
          <span className="w-4 shrink-0" />
        )}
        <span>{highlightText(node.label, searchText)}</span>
        {node.level != null && node.level !== 0 && (
          <LevelBadge level={node.level} config={config} />
        )}
      </div>
      {expanded && hasChildren && (
        <div className="ml-4">
          {filteredChildren.map((child) => (
            <ClassificationTreeNode
              key={child.key}
              node={child}
              config={config}
              searchText={searchText}
              defaultExpanded={defaultExpanded}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function ClassificationTree({
  config,
}: {
  config: DataClassificationSetting_DataClassificationConfig;
}) {
  const { t } = useTranslation();
  const [searchText, setSearchText] = useState("");
  const treeData = useMemo(() => buildTreeData(config), [config]);

  const filteredTree = useMemo(
    () => treeData.filter((n) => matchesSearch(n, searchText)),
    [treeData, searchText]
  );

  return (
    <div className="space-y-4 h-full">
      <div>
        <SearchInput
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          placeholder={t("schema-template.classification.search")}
        />
      </div>
      <div>
        {filteredTree.map((node) => (
          <ClassificationTreeNode
            key={node.key}
            node={node}
            config={config}
            searchText={searchText}
            defaultExpanded={true}
          />
        ))}
      </div>
    </div>
  );
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
          <a
            href="https://docs.bytebase.com/security/data-masking/data-classification?source=console"
            target="_blank"
            rel="noopener noreferrer"
            className="ml-1 inline-flex items-center gap-0.5 text-accent hover:underline"
          >
            {t("common.learn-more")}
            <ExternalLink className="w-3 h-3" />
          </a>
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
                  disabled={!editorDirty || !hasClassificationFeature}
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
          <div className="text-sm text-control-light space-y-1">
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
