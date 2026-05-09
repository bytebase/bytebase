import { create, fromJsonString, toJsonString } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { Pencil, X } from "lucide-react";
import type { MouseEvent, ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  updateColumnCatalog,
  updateTableCatalog,
} from "@/components/ColumnDataTable/utils";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { SearchInput } from "@/react/components/ui/search-input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { Textarea } from "@/react/components/ui/textarea";
import { useVueState } from "@/react/hooks/useVueState";
import {
  getTableCatalog,
  pushNotification,
  useDatabaseCatalog,
  useDatabaseCatalogV1Store,
  useSettingV1Store,
  useSubscriptionV1Store,
} from "@/store";
import {
  SchemaCatalogSchema,
  TableCatalogSchema,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type {
  DataClassificationSetting_DataClassificationConfig,
  SemanticTypeSetting_SemanticType,
} from "@/types/proto-es/v1/setting_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  getDatabaseProject,
  getInstanceResource,
  hasWorkspacePermissionV2,
  instanceV1MaskingForNoSQL,
} from "@/utils";

interface TableColumnDetail {
  characterSet?: string;
  classification?: string;
  collation?: string;
  comment?: string;
  defaultValue: string;
  name: string;
  nullable: boolean;
  semanticType?: string;
  type: string;
}

interface TablePartitionDetail {
  children: TablePartitionDetail[];
  expression: string;
  name: string;
  type: string;
}

interface TriggerDetail {
  body: string;
  event: string;
  name: string;
  sqlMode?: string;
  timing: string;
}

export interface TableDetailDialogData {
  classification?: string;
  classificationConfig?: DataClassificationSetting_DataClassificationConfig;
  columns: TableColumnDetail[];
  collation?: string;
  dataSize: string;
  database?: Database;
  editable?: boolean;
  engine?: string;
  indexes: {
    comment?: string;
    expressions: string[];
    name: string;
    unique: boolean;
    visible?: boolean;
  }[];
  indexSize: string;
  name: string;
  rowCount: string;
  schema?: string;
  showCharacterSet?: boolean;
  showColumnClassification?: boolean;
  showColumnCollation?: boolean;
  showColumns?: boolean;
  showCollation?: boolean;
  showEngine?: boolean;
  showIndexComment?: boolean;
  showIndexes?: boolean;
  showIndexSize?: boolean;
  showIndexVisible?: boolean;
  showPartitionTables?: boolean;
  showSemanticType?: boolean;
  showTriggers?: boolean;
  tableName?: string;
  partitions?: TablePartitionDetail[];
  triggers?: TriggerDetail[];
}

interface ClassificationTreeNode {
  children: ClassificationTreeNode[];
  id: string;
  isLeaf: boolean;
  level?: number;
  title: string;
}

const bgColorList = [
  "bg-green-200",
  "bg-yellow-200",
  "bg-orange-300",
  "bg-amber-500",
  "bg-red-500",
];

function toTestId(value: string) {
  return value.replace(/[^a-zA-Z0-9_-]/g, "-");
}

function getClassificationEntry(
  classification: string | undefined,
  classificationConfig:
    | DataClassificationSetting_DataClassificationConfig
    | undefined
) {
  if (!classification || !classificationConfig) {
    return undefined;
  }

  return classificationConfig.classification[classification];
}

function sortClassification(item1: { id: string }, item2: { id: string }) {
  const id1s = item1.id.split("-");
  const id2s = item2.id.split("-");

  if (id1s.length !== id2s.length) {
    return id1s.length - id2s.length;
  }

  for (let i = 0; i < id1s.length; i++) {
    if (id1s[i] === id2s[i]) {
      continue;
    }

    const id1 = Number(id1s[i]);
    const id2 = Number(id2s[i]);
    if (Number.isNaN(id1) || Number.isNaN(id2)) {
      return id1s[i].localeCompare(id2s[i]);
    }
    return id1 - id2;
  }

  return 0;
}

function buildClassificationTree(
  classificationConfig: DataClassificationSetting_DataClassificationConfig
) {
  const classifications = Object.values(
    classificationConfig.classification
  ).sort(sortClassification);
  const nodeMap = new Map<string, ClassificationTreeNode>();
  const roots: ClassificationTreeNode[] = [];

  for (const classification of classifications) {
    const node: ClassificationTreeNode = {
      id: classification.id,
      title: classification.title,
      level: classification.level,
      children: [],
      isLeaf: true,
    };
    nodeMap.set(classification.id, node);
  }

  for (const classification of classifications) {
    const node = nodeMap.get(classification.id);
    if (!node) continue;

    const parts = classification.id.split("-");
    if (parts.length === 1) {
      roots.push(node);
      continue;
    }

    const parentId = parts.slice(0, -1).join("-");
    const parent = nodeMap.get(parentId);
    if (!parent) {
      roots.push(node);
      continue;
    }

    parent.children.push(node);
    parent.isLeaf = false;
  }

  const sortNodes = (nodes: ClassificationTreeNode[]) => {
    nodes.sort(sortClassification);
    for (const node of nodes) {
      sortNodes(node.children);
      node.isLeaf = node.children.length === 0;
    }
  };

  sortNodes(roots);
  return roots;
}

function filterClassificationTree(
  nodes: ClassificationTreeNode[],
  searchText: string
): ClassificationTreeNode[] {
  const normalizedSearch = searchText.trim().toLowerCase();
  if (!normalizedSearch) {
    return nodes;
  }

  return nodes
    .map((node) => {
      const children = filterClassificationTree(
        node.children,
        normalizedSearch
      );
      const label = `${node.id} ${node.title}`.toLowerCase();
      if (label.includes(normalizedSearch) || children.length > 0) {
        return {
          ...node,
          children,
          isLeaf: children.length === 0 && node.children.length === 0,
        };
      }
      return undefined;
    })
    .filter((node): node is ClassificationTreeNode => node !== undefined);
}

function flattenClassificationTree(
  nodes: ClassificationTreeNode[],
  depth = 0
): Array<{ depth: number; node: ClassificationTreeNode }> {
  return nodes.flatMap((node) => [
    { node, depth },
    ...flattenClassificationTree(node.children, depth + 1),
  ]);
}

function getSemanticTypeTitle(
  semanticTypeId: string | undefined,
  semanticTypeList: SemanticTypeSetting_SemanticType[]
) {
  if (!semanticTypeId) {
    return "";
  }

  if (!hasWorkspacePermissionV2("bb.settings.get")) {
    return semanticTypeId;
  }

  return (
    semanticTypeList.find((semanticType) => semanticType.id === semanticTypeId)
      ?.title || semanticTypeId
  );
}

function MiniActionButton({
  ariaLabel,
  children,
  className,
  dataTestId,
  disabled,
  onClick,
}: {
  ariaLabel: string;
  children: ReactNode;
  className?: string;
  dataTestId?: string;
  disabled?: boolean;
  onClick: (event: MouseEvent<HTMLButtonElement>) => void;
}) {
  return (
    <button
      type="button"
      aria-label={ariaLabel}
      data-testid={dataTestId}
      className={`inline-flex size-5 items-center justify-center rounded-xs text-control transition-colors hover:bg-control-bg hover:text-main disabled:cursor-not-allowed disabled:opacity-50 [&_svg]:pointer-events-none ${className ?? ""}`}
      disabled={disabled}
      onKeyDown={(event) => event.stopPropagation()}
      onMouseDown={(event) => event.stopPropagation()}
      onPointerDown={(event) => event.stopPropagation()}
      onClick={onClick}
    >
      {children}
    </button>
  );
}

function ClassificationLevelBadge({
  classification,
  classificationConfig,
  placeholder = "N/A",
  showText = true,
}: {
  classification?: string;
  classificationConfig?: DataClassificationSetting_DataClassificationConfig;
  placeholder?: string;
  showText?: boolean;
}) {
  const classificationEntry = getClassificationEntry(
    classification,
    classificationConfig
  );
  const level = (classificationConfig?.levels ?? []).find(
    (level) => level.level === classificationEntry?.level
  );
  const levelColor =
    bgColorList[(classificationEntry?.level ?? 0) - 1] ?? "bg-control-bg-hover";

  return (
    <span className="flex min-w-0 items-center gap-x-1">
      {showText && (
        <span className="min-w-0 truncate">
          {classificationEntry?.title || classification || (
            <span className="text-control-placeholder italic">
              {placeholder}
            </span>
          )}
        </span>
      )}
      {level && (
        <span className={`rounded px-1 py-0.5 text-xs ${levelColor}`}>
          {level.title}
        </span>
      )}
    </span>
  );
}

function DetailSection({
  children,
  title,
}: {
  children: ReactNode;
  title: string;
}) {
  return (
    <div className="mt-6 flex flex-col gap-y-4">
      <div className="text-sm font-medium text-control-light">{title}</div>
      {children}
    </div>
  );
}

function NoSQLCatalogEditor({
  database,
  readonly,
  schema,
  tableName,
}: {
  database: Database;
  readonly: boolean;
  schema: string;
  tableName: string;
}) {
  const { t } = useTranslation();
  const databaseCatalogStore = useDatabaseCatalogV1Store();
  const databaseCatalog = useDatabaseCatalog(database.name, false);
  const catalog = useVueState(() => databaseCatalog.value);
  const [catalogText, setCatalogText] = useState("{}");
  const [isUploading, setIsUploading] = useState(false);

  const initialCatalogText = useMemo(() => {
    return toJsonString(
      TableCatalogSchema,
      getTableCatalog(catalog, schema, tableName) ??
        create(TableCatalogSchema, {
          name: tableName,
        }),
      {
        prettySpaces: 2,
      }
    );
  }, [catalog, schema, tableName]);

  useEffect(() => {
    setCatalogText(initialCatalogText);
  }, [initialCatalogText]);

  const hasChanges = catalogText !== initialCatalogText;

  const handleUpload = async () => {
    const nextTableCatalog = fromJsonString(TableCatalogSchema, catalogText);
    if (nextTableCatalog.name !== tableName) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: `catalog name must be ${tableName}`,
      });
      return;
    }

    setIsUploading(true);
    try {
      const pendingCatalog = cloneDeep(catalog);
      const schemaCatalog = pendingCatalog.schemas.find(
        (schemaCatalog) => schemaCatalog.name === schema
      );
      if (schemaCatalog) {
        const tableIndex = schemaCatalog.tables.findIndex(
          (tableCatalog) => tableCatalog.name === tableName
        );
        if (tableIndex >= 0) {
          schemaCatalog.tables[tableIndex] = nextTableCatalog;
        } else {
          schemaCatalog.tables.push(nextTableCatalog);
        }
      } else {
        pendingCatalog.schemas.push(
          create(SchemaCatalogSchema, {
            name: schema,
            tables: [nextTableCatalog],
          })
        );
      }

      await databaseCatalogStore.updateDatabaseCatalog(pendingCatalog);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <DetailSection title={t("common.catalog")}>
      <div className="text-sm text-control-light">
        {t("db.catalog.description")}{" "}
        <a
          className="normal-link"
          href="https://api.bytebase.com/#tag/databasecatalogservice/PATCH/v1/instances/{instance}/databases/{database}/catalog"
          rel="noreferrer"
          target="_blank"
        >
          {t("common.view-doc")}
        </a>
      </div>
      <Textarea
        data-testid="nosql-catalog-editor"
        className="min-h-80 font-mono"
        disabled={readonly}
        value={catalogText}
        onChange={(event) => setCatalogText(event.target.value)}
      />
      <div className="flex justify-end gap-x-2">
        <Button
          data-testid="nosql-catalog-upload"
          disabled={readonly || !hasChanges || isUploading}
          onClick={() => void handleUpload()}
        >
          {isUploading ? t("common.updating") : t("common.upload")}
        </Button>
      </div>
    </DetailSection>
  );
}

function flattenPartitionRows(
  partitions: TablePartitionDetail[],
  depth = 0
): Array<{ depth: number; partition: TablePartitionDetail }> {
  return partitions.flatMap((partition) => [
    { depth, partition },
    ...flattenPartitionRows(partition.children, depth + 1),
  ]);
}

function ClassificationPickerDialog({
  classificationConfig,
  onOpenChange,
  onSelect,
  open,
}: {
  classificationConfig: DataClassificationSetting_DataClassificationConfig;
  onOpenChange: (open: boolean) => void;
  onSelect: (classificationId: string) => void;
  open: boolean;
}) {
  const { t } = useTranslation();
  const [searchText, setSearchText] = useState("");

  useEffect(() => {
    if (open) {
      setSearchText("");
    }
  }, [open]);

  const rows = useMemo(() => {
    const tree = buildClassificationTree(classificationConfig);
    return flattenClassificationTree(
      filterClassificationTree(tree, searchText)
    );
  }, [classificationConfig, searchText]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl p-6">
        <DialogTitle>{t("schema-template.classification.select")}</DialogTitle>
        <div className="mt-4 flex flex-col gap-y-4">
          <SearchInput
            placeholder={t("schema-template.classification.search")}
            value={searchText}
            onChange={(event) => setSearchText(event.target.value)}
          />
          <div className="max-h-[28rem] overflow-y-auto rounded-lg border border-block-border">
            {rows.length === 0 ? (
              <div className="px-4 py-6 text-sm text-control-light">
                {t("common.no-data")}
              </div>
            ) : (
              <div className="divide-y divide-block-border">
                {rows.map(({ node, depth }) => {
                  const content = (
                    <>
                      <span className="truncate">
                        {node.id} {node.title}
                      </span>
                      {node.level !== undefined && (
                        <ClassificationLevelBadge
                          classification={node.id}
                          classificationConfig={classificationConfig}
                          showText={false}
                        />
                      )}
                    </>
                  );

                  if (!node.isLeaf) {
                    return (
                      <div
                        key={node.id}
                        className="flex items-center gap-x-2 px-4 py-2 text-sm font-medium text-main"
                        style={{ paddingLeft: `${depth * 16 + 16}px` }}
                      >
                        {content}
                      </div>
                    );
                  }

                  return (
                    <button
                      key={node.id}
                      type="button"
                      data-testid={`classification-option-${toTestId(node.id)}`}
                      className="flex w-full items-center justify-between gap-x-2 px-4 py-2 text-left text-sm text-control hover:bg-control-bg"
                      style={{ paddingLeft: `${depth * 16 + 16}px` }}
                      onClick={() => {
                        onSelect(node.id);
                        onOpenChange(false);
                      }}
                    >
                      {content}
                    </button>
                  );
                })}
              </div>
            )}
          </div>
          <div className="flex justify-end gap-x-2">
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              {t("common.cancel")}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function SemanticTypePickerDialog({
  onOpenChange,
  onSelect,
  open,
  semanticTypeList,
}: {
  onOpenChange: (open: boolean) => void;
  onSelect: (semanticTypeId: string) => void;
  open: boolean;
  semanticTypeList: SemanticTypeSetting_SemanticType[];
}) {
  const { t } = useTranslation();
  const [searchText, setSearchText] = useState("");

  useEffect(() => {
    if (open) {
      setSearchText("");
    }
  }, [open]);

  const filteredSemanticTypeList = useMemo(() => {
    const normalizedSearch = searchText.trim().toLowerCase();
    if (!normalizedSearch) {
      return semanticTypeList;
    }

    return semanticTypeList.filter((semanticType) => {
      return (
        semanticType.id.toLowerCase().includes(normalizedSearch) ||
        semanticType.title.toLowerCase().includes(normalizedSearch) ||
        (semanticType.description?.toLowerCase().includes(normalizedSearch) ??
          false)
      );
    });
  }, [searchText, semanticTypeList]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl p-6">
        <DialogTitle>
          {t("settings.sensitive-data.semantic-types.self")}
        </DialogTitle>
        <div className="mt-4 flex flex-col gap-y-4">
          <SearchInput
            placeholder={t("common.filter-by-name")}
            value={searchText}
            onChange={(event) => setSearchText(event.target.value)}
          />
          <div className="max-h-[28rem] overflow-y-auto rounded-lg border border-block-border">
            {filteredSemanticTypeList.length === 0 ? (
              <div className="px-4 py-6 text-sm text-control-light">
                {t("common.no-data")}
              </div>
            ) : (
              <div className="divide-y divide-block-border">
                {filteredSemanticTypeList.map((semanticType) => (
                  <button
                    key={semanticType.id}
                    type="button"
                    data-testid={`semantic-type-option-${toTestId(semanticType.id)}`}
                    className="flex w-full flex-col items-start gap-y-1 px-4 py-3 text-left hover:bg-control-bg"
                    onClick={() => {
                      onSelect(semanticType.id);
                      onOpenChange(false);
                    }}
                  >
                    <span className="text-sm font-medium text-main">
                      {semanticType.title}
                    </span>
                    <span className="text-xs text-control">
                      {semanticType.id}
                    </span>
                    {semanticType.description && (
                      <span className="text-xs text-control-light">
                        {semanticType.description}
                      </span>
                    )}
                  </button>
                ))}
              </div>
            )}
          </div>
          <div className="flex justify-end gap-x-2">
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              {t("common.cancel")}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export function EditableClassificationCell({
  classification,
  classificationConfig,
  onApply,
  readonly = false,
  testIdPrefix,
}: {
  classification?: string;
  classificationConfig?: DataClassificationSetting_DataClassificationConfig;
  onApply: (classificationId: string) => Promise<void> | void;
  readonly?: boolean;
  testIdPrefix: string;
}) {
  const { t } = useTranslation();
  const [showPicker, setShowPicker] = useState(false);

  return (
    <>
      <div className="flex min-w-0 items-center gap-x-1">
        <ClassificationLevelBadge
          classification={classification}
          classificationConfig={classificationConfig}
        />
        {!readonly && classification && (
          <MiniActionButton
            ariaLabel={t("common.remove")}
            dataTestId={`${testIdPrefix}-remove`}
            onClick={(event) => {
              event.stopPropagation();
              void onApply("");
            }}
          >
            <X className="size-3" />
          </MiniActionButton>
        )}
        {!readonly && classificationConfig && (
          <MiniActionButton
            ariaLabel={t("common.edit")}
            dataTestId={`${testIdPrefix}-edit`}
            onClick={(event) => {
              event.stopPropagation();
              setShowPicker(true);
            }}
          >
            <Pencil className="size-3" />
          </MiniActionButton>
        )}
      </div>
      {classificationConfig && (
        <ClassificationPickerDialog
          classificationConfig={classificationConfig}
          open={showPicker}
          onOpenChange={setShowPicker}
          onSelect={(classificationId) => {
            void onApply(classificationId);
          }}
        />
      )}
    </>
  );
}

function EditableSemanticTypeCell({
  database,
  onApply,
  readonly = false,
  semanticTypeId,
  testIdPrefix,
}: {
  database: Database;
  onApply: (semanticTypeId: string) => Promise<void> | void;
  readonly?: boolean;
  semanticTypeId?: string;
  testIdPrefix: string;
}) {
  const { t } = useTranslation();
  const settingStore = useSettingV1Store();
  const subscriptionStore = useSubscriptionV1Store();
  const [showFeatureDialog, setShowFeatureDialog] = useState(false);
  const [showPicker, setShowPicker] = useState(false);

  const semanticTypeList = useVueState(() => {
    const setting = settingStore.getSettingByName(
      Setting_SettingName.SEMANTIC_TYPES
    );
    return setting?.value?.value.case === "semanticType"
      ? ((setting.value.value.value.types ??
          []) as SemanticTypeSetting_SemanticType[])
      : [];
  });
  const hasSensitiveDataFeature = useVueState(() =>
    subscriptionStore.hasFeature(PlanFeature.FEATURE_DATA_MASKING)
  );
  const instanceMissingLicense = useVueState(() =>
    subscriptionStore.instanceMissingLicense(
      PlanFeature.FEATURE_DATA_MASKING,
      getInstanceResource(database)
    )
  );

  useEffect(() => {
    void settingStore.getOrFetchSettingByName(
      Setting_SettingName.SEMANTIC_TYPES,
      true
    );
  }, [settingStore]);

  const semanticTypeTitle = getSemanticTypeTitle(
    semanticTypeId,
    semanticTypeList
  );

  return (
    <>
      <div className="flex min-w-0 items-center gap-x-1">
        <span className="min-w-0 truncate">
          {semanticTypeTitle || (
            <span className="text-control-placeholder italic">N/A</span>
          )}
        </span>
        {!readonly && semanticTypeId && (
          <MiniActionButton
            ariaLabel={t("common.remove")}
            dataTestId={`${testIdPrefix}-remove`}
            onClick={(event) => {
              event.stopPropagation();
              void onApply("");
            }}
          >
            <X className="size-3" />
          </MiniActionButton>
        )}
        {!readonly && (
          <MiniActionButton
            ariaLabel={t("common.edit")}
            dataTestId={`${testIdPrefix}-edit`}
            onClick={(event) => {
              event.stopPropagation();
              if (!hasSensitiveDataFeature || instanceMissingLicense) {
                setShowFeatureDialog(true);
                return;
              }
              setShowPicker(true);
            }}
          >
            <Pencil className="size-3" />
          </MiniActionButton>
        )}
      </div>

      <SemanticTypePickerDialog
        open={showPicker}
        semanticTypeList={semanticTypeList}
        onOpenChange={setShowPicker}
        onSelect={(nextSemanticTypeId) => {
          void onApply(nextSemanticTypeId);
        }}
      />

      <Dialog open={showFeatureDialog} onOpenChange={setShowFeatureDialog}>
        <DialogContent className="p-6">
          <DialogTitle>{t("common.warning")}</DialogTitle>
          <FeatureAttention
            feature={PlanFeature.FEATURE_DATA_MASKING}
            instance={getInstanceResource(database)}
          />
          <div className="mt-4 flex justify-end gap-x-2">
            <Button
              variant="outline"
              onClick={() => setShowFeatureDialog(false)}
            >
              {t("common.cancel")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function TableDetailDialog({
  table,
  open,
  onOpenChange,
}: {
  table?: TableDetailDialogData;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useTranslation();
  const settingStore = useSettingV1Store();
  const [columnSearchKeyword, setColumnSearchKeyword] = useState("");

  const filteredColumns = useMemo(() => {
    if (!table) {
      return [];
    }

    const keyword = columnSearchKeyword.trim().toLowerCase();
    if (!keyword) {
      return table.columns;
    }

    return table.columns.filter((column) =>
      column.name.toLowerCase().includes(keyword)
    );
  }, [columnSearchKeyword, table?.columns]);

  useEffect(() => {
    setColumnSearchKeyword("");
  }, [table?.name]);

  const classificationConfig = useVueState(() => {
    if (table?.classificationConfig) {
      return table.classificationConfig;
    }
    if (!table?.database) {
      return undefined;
    }
    return settingStore.getProjectClassification(
      getDatabaseProject(table.database).dataClassificationConfigId ?? ""
    );
  });

  useEffect(() => {
    void settingStore.getOrFetchSettingByName(
      Setting_SettingName.DATA_CLASSIFICATION,
      true
    );
  }, [settingStore]);

  if (!table) {
    return null;
  }

  const showCharacterSetColumn = table.showCharacterSet;
  const showColumnClassification = table.showColumnClassification;
  const showColumnCollation = table.showColumnCollation;
  const showColumns = table.showColumns ?? true;
  const showNoSQLCatalog =
    !!table.database &&
    instanceV1MaskingForNoSQL(getInstanceResource(table.database));
  const showIndexCommentColumn = table.showIndexComment;
  const showIndexVisibleColumn = table.showIndexVisible;
  const showPartitionTables =
    table.showPartitionTables && (table.partitions?.length ?? 0) > 0;
  const showSemanticTypeColumn = table.showSemanticType;
  const showSummaryCollation = table.showCollation;
  const showTriggers = table.showTriggers && (table.triggers?.length ?? 0) > 0;
  const readonly = !table.editable;
  const partitionRows = flattenPartitionRows(table.partitions ?? []);

  const handleTableClassificationApply = async (classificationId: string) => {
    if (!table.database || !table.tableName) {
      return;
    }

    await updateTableCatalog({
      database: table.database.name,
      schema: table.schema ?? "",
      table: table.tableName,
      tableCatalog: {
        classification: classificationId,
      },
    });
  };

  const handleColumnClassificationApply = async (
    columnName: string,
    classificationId: string
  ) => {
    if (!table.database || !table.tableName) {
      return;
    }

    await updateColumnCatalog({
      database: table.database.name,
      schema: table.schema ?? "",
      table: table.tableName,
      column: columnName,
      columnCatalog: {
        classification: classificationId,
      },
      notification: !classificationId
        ? t("common.removed")
        : t("common.update"),
    });
  };

  const handleColumnSemanticTypeApply = async (
    columnName: string,
    semanticTypeId: string
  ) => {
    if (!table.database || !table.tableName) {
      return;
    }

    await updateColumnCatalog({
      database: table.database.name,
      schema: table.schema ?? "",
      table: table.tableName,
      column: columnName,
      columnCatalog: {
        semanticType: semanticTypeId,
      },
      notification: !semanticTypeId ? t("common.removed") : t("common.update"),
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl p-6">
        <DialogTitle>{table.name}</DialogTitle>
        <div className="mt-4 grid gap-4 sm:grid-cols-3">
          <div>
            <div className="text-sm font-medium text-control-light">
              {t("database.classification.self")}
            </div>
            <div className="mt-1 text-sm text-main">
              <EditableClassificationCell
                classification={table.classification}
                classificationConfig={classificationConfig}
                readonly={readonly}
                testIdPrefix="table-classification"
                onApply={handleTableClassificationApply}
              />
            </div>
          </div>
          {table.showEngine && (
            <div>
              <div className="text-sm font-medium text-control-light">
                {t("database.engine")}
              </div>
              <div className="mt-1 text-sm text-main">
                {table.engine || "-"}
              </div>
            </div>
          )}
          <div>
            <div className="text-sm font-medium text-control-light">
              {t("database.row-count-estimate")}
            </div>
            <div className="mt-1 text-sm text-main">{table.rowCount}</div>
          </div>
          <div>
            <div className="text-sm font-medium text-control-light">
              {t("database.data-size")}
            </div>
            <div className="mt-1 text-sm text-main">{table.dataSize}</div>
          </div>
          {table.showIndexSize && (
            <div>
              <div className="text-sm font-medium text-control-light">
                {t("database.index-size")}
              </div>
              <div className="mt-1 text-sm text-main">{table.indexSize}</div>
            </div>
          )}
          {showSummaryCollation && (
            <div>
              <div className="text-sm font-medium text-control-light">
                {t("db.collation")}
              </div>
              <div className="mt-1 text-sm text-main">
                {table.collation || "-"}
              </div>
            </div>
          )}
        </div>

        {showPartitionTables && (
          <DetailSection title={t("database.partition-tables")}>
            <div className="rounded-lg border border-block-border">
              <Table className="min-w-full">
                <TableHeader className="bg-control-bg">
                  <TableRow className="hover:bg-control-bg">
                    <TableHead>{t("common.name")}</TableHead>
                    <TableHead>{t("common.type")}</TableHead>
                    <TableHead>{t("database.expression")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody className="bg-background">
                  {partitionRows.map(({ depth, partition }) => (
                    <TableRow key={`${partition.name}-${depth}`}>
                      <TableCell className="text-main">
                        <span style={{ paddingLeft: `${depth * 16}px` }}>
                          {partition.name}
                        </span>
                      </TableCell>
                      <TableCell>{partition.type}</TableCell>
                      <TableCell>{partition.expression || "-"}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </DetailSection>
        )}

        {showColumns && (
          <DetailSection title={t("database.columns")}>
            <div className="flex items-center justify-between gap-3">
              <Input
                className="w-full max-w-sm"
                placeholder={t("common.filter-by-name")}
                value={columnSearchKeyword}
                onChange={(event) => setColumnSearchKeyword(event.target.value)}
              />
            </div>
            <div className="rounded-lg border border-block-border">
              <table className="min-w-full divide-y divide-block-border text-sm">
                <thead className="bg-control-bg">
                  <tr className="text-left text-sm text-control-light">
                    <th className="px-4 py-2 font-medium">
                      {t("common.name")}
                    </th>
                    {showSemanticTypeColumn && (
                      <th className="px-4 py-2 font-medium">
                        {t(
                          "settings.sensitive-data.semantic-types.table.semantic-type"
                        )}
                      </th>
                    )}
                    {showColumnClassification && (
                      <th className="px-4 py-2 font-medium">
                        {t("database.classification.self")}
                      </th>
                    )}
                    <th className="px-4 py-2 font-medium">
                      {t("common.type")}
                    </th>
                    <th className="px-4 py-2 font-medium">
                      {t("common.default")}
                    </th>
                    <th className="px-4 py-2 font-medium">
                      {t("database.nullable")}
                    </th>
                    {showCharacterSetColumn && (
                      <th className="px-4 py-2 font-medium">
                        {t("db.character-set")}
                      </th>
                    )}
                    {showColumnCollation && (
                      <th className="px-4 py-2 font-medium">
                        {t("db.collation")}
                      </th>
                    )}
                    <th className="px-4 py-2 font-medium">
                      {t("common.comment")}
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-block-border bg-background">
                  {filteredColumns.map((column) => (
                    <tr key={column.name}>
                      <td className="px-4 py-3 text-sm text-main">
                        {column.name}
                      </td>
                      {showSemanticTypeColumn && (
                        <td className="px-4 py-3 text-sm text-control">
                          {table.database ? (
                            <EditableSemanticTypeCell
                              database={table.database}
                              readonly={readonly}
                              semanticTypeId={column.semanticType}
                              testIdPrefix={`column-semantic-type-${toTestId(column.name)}`}
                              onApply={(semanticTypeId) =>
                                handleColumnSemanticTypeApply(
                                  column.name,
                                  semanticTypeId
                                )
                              }
                            />
                          ) : (
                            column.semanticType || "-"
                          )}
                        </td>
                      )}
                      {showColumnClassification && (
                        <td className="px-4 py-3 text-sm text-control">
                          <EditableClassificationCell
                            classification={column.classification}
                            classificationConfig={classificationConfig}
                            readonly={readonly}
                            testIdPrefix={`column-classification-${toTestId(column.name)}`}
                            onApply={(classificationId) =>
                              handleColumnClassificationApply(
                                column.name,
                                classificationId
                              )
                            }
                          />
                        </td>
                      )}
                      <td className="px-4 py-3 text-sm text-control">
                        {column.type}
                      </td>
                      <td className="px-4 py-3 text-sm text-control">
                        {column.defaultValue}
                      </td>
                      <td className="px-4 py-3 text-sm text-control">
                        <Checkbox checked={column.nullable} disabled />
                      </td>
                      {showCharacterSetColumn && (
                        <td className="px-4 py-3 text-sm text-control">
                          {column.characterSet || "-"}
                        </td>
                      )}
                      {showColumnCollation && (
                        <td className="px-4 py-3 text-sm text-control">
                          {column.collation || "-"}
                        </td>
                      )}
                      <td className="px-4 py-3 text-sm text-control">
                        {column.comment || "-"}
                      </td>
                    </tr>
                  ))}
                  {filteredColumns.length === 0 && (
                    <tr>
                      <td
                        className="px-4 py-6 text-center text-sm text-control-light"
                        colSpan={
                          5 +
                          (showSemanticTypeColumn ? 1 : 0) +
                          (showColumnClassification ? 1 : 0) +
                          (showCharacterSetColumn ? 1 : 0) +
                          (showColumnCollation ? 1 : 0)
                        }
                      >
                        {t("common.no-data")}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </DetailSection>
        )}

        {table.showIndexes && table.indexes.length > 0 && (
          <DetailSection title={t("database.indexes")}>
            {table.indexes.map((index) => (
              <div
                key={index.name}
                className="rounded-lg border border-block-border"
              >
                <div className="border-b border-block-border px-4 py-3 text-base font-medium text-main">
                  {index.name}
                </div>
                <table className="min-w-full divide-y divide-block-border text-sm">
                  <thead className="bg-control-bg">
                    <tr className="text-left text-sm text-control-light">
                      <th className="px-4 py-2 font-medium">
                        {t("database.expression")}
                      </th>
                      <th className="px-4 py-2 font-medium">
                        {t("database.unique")}
                      </th>
                      {showIndexVisibleColumn && (
                        <th className="px-4 py-2 font-medium">
                          {t("database.visible")}
                        </th>
                      )}
                      {showIndexCommentColumn && (
                        <th className="px-4 py-2 font-medium">
                          {t("common.comment")}
                        </th>
                      )}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-block-border bg-background">
                    <tr>
                      <td className="px-4 py-3 text-sm text-control">
                        {index.expressions.join(", ") || "-"}
                      </td>
                      <td className="px-4 py-3 text-sm text-control">
                        {String(index.unique)}
                      </td>
                      {showIndexVisibleColumn && (
                        <td className="px-4 py-3 text-sm text-control">
                          {String(index.visible)}
                        </td>
                      )}
                      {showIndexCommentColumn && (
                        <td className="px-4 py-3 text-sm text-control">
                          {index.comment || "-"}
                        </td>
                      )}
                    </tr>
                  </tbody>
                </table>
              </div>
            ))}
          </DetailSection>
        )}

        {showTriggers && (
          <DetailSection title={t("db.triggers")}>
            <div className="rounded-lg border border-block-border">
              <Table className="min-w-full">
                <TableHeader className="bg-control-bg">
                  <TableRow className="hover:bg-control-bg">
                    <TableHead>{t("common.name")}</TableHead>
                    <TableHead>{t("db.trigger.event")}</TableHead>
                    <TableHead>{t("db.trigger.timing")}</TableHead>
                    <TableHead>{t("db.trigger.body")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody className="bg-background">
                  {table.triggers?.map((trigger) => (
                    <TableRow key={trigger.name}>
                      <TableCell className="text-main">
                        {trigger.name}
                      </TableCell>
                      <TableCell>{trigger.event || "-"}</TableCell>
                      <TableCell>{trigger.timing || "-"}</TableCell>
                      <TableCell className="max-w-xl whitespace-pre-wrap break-all">
                        {trigger.body || trigger.sqlMode || "-"}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </DetailSection>
        )}

        {showNoSQLCatalog && table.database && table.tableName && (
          <NoSQLCatalogEditor
            database={table.database}
            readonly={readonly}
            schema={table.schema ?? ""}
            tableName={table.tableName}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}
