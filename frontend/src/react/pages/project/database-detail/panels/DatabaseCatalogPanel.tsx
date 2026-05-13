import { create } from "@bufbuild/protobuf";
import { ShieldCheck } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import type {
  MaskData,
  MaskDataTarget,
} from "@/react/lib/sensitive-data/types";
import {
  getMaskDataIdentifier,
  isCurrentColumnException,
} from "@/react/lib/sensitive-data/utils";
import {
  featureToRef,
  pushNotification,
  useDatabaseCatalog,
  useDatabaseCatalogV1Store,
  usePolicyV1Store,
  useSettingV1Store,
} from "@/store";
import type { Permission } from "@/types";
import type {
  ColumnCatalog,
  DatabaseCatalog,
  ObjectSchema,
  TableCatalog,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import { ObjectSchema_Type } from "@/types/proto-es/v1/database_catalog_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  MaskingExemptionPolicySchema,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  type SemanticTypeSetting_SemanticType,
  Setting_SettingName,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  getDatabaseProject,
  getInstanceResource,
  hasProjectPermissionV2,
  instanceV1MaskingForNoSQL,
} from "@/utils";
import { GrantAccessDialog } from "../catalog/GrantAccessDialog";
import { SensitiveColumnTable } from "../catalog/SensitiveColumnTable";

const GRANT_ACCESS_PERMISSIONS: Permission[] = [
  "bb.policies.createMaskingExemptionPolicy",
  "bb.policies.updateMaskingExemptionPolicy",
  "bb.databaseCatalogs.get",
];

const flattenObjectSchema = (
  parentPath: string,
  objectSchema: ObjectSchema
): {
  column: string;
  semanticTypeId: string;
  target: ObjectSchema;
}[] => {
  switch (objectSchema.type) {
    case ObjectSchema_Type.OBJECT: {
      const result: {
        column: string;
        semanticTypeId: string;
        target: ObjectSchema;
      }[] = [];
      if (objectSchema.kind?.case === "structKind") {
        for (const [key, schema] of Object.entries(
          objectSchema.kind.value.properties ?? {}
        )) {
          result.push(
            ...flattenObjectSchema(
              [parentPath, key].filter((item) => item).join("."),
              schema
            )
          );
        }
      }
      return result;
    }
    case ObjectSchema_Type.ARRAY:
      if (
        objectSchema.kind?.case === "arrayKind" &&
        objectSchema.kind.value.kind
      ) {
        return flattenObjectSchema(parentPath, objectSchema.kind.value.kind);
      }
      return [];
    default:
      return [
        {
          column: parentPath,
          semanticTypeId: objectSchema.semanticType,
          target: objectSchema,
        },
      ];
  }
};

const flattenSensitiveColumnList = (catalog?: DatabaseCatalog): MaskData[] => {
  if (!catalog) {
    return [];
  }

  const result: MaskData[] = [];

  for (const schema of catalog.schemas) {
    for (const table of schema.tables) {
      if (table.kind?.case === "columns") {
        for (const column of table.kind.value.columns ?? []) {
          if (!column.semanticType && !column.classification) {
            continue;
          }
          result.push({
            schema: schema.name,
            table: table.name,
            column: column.name,
            semanticTypeId: column.semanticType,
            classificationId: column.classification,
            target: column,
          });
        }
      }

      if (table.kind?.case === "objectSchema") {
        const flattened = flattenObjectSchema("", table.kind.value);
        result.push(
          ...flattened.map((item) => ({
            ...item,
            schema: schema.name,
            table: table.name,
            classificationId: "",
            disableClassification: true,
          }))
        );
      }

      if (table.classification) {
        result.push({
          schema: schema.name,
          table: table.name,
          column: "",
          semanticTypeId: "",
          disableSemanticType: true,
          classificationId: table.classification,
          target: table,
        });
      }
    }
  }

  return result;
};

const hasSemanticType = (
  target: MaskDataTarget
): target is ColumnCatalog | ObjectSchema => {
  return "semanticType" in target;
};

const hasClassificationType = (
  target: MaskDataTarget
): target is ColumnCatalog | TableCatalog => {
  return "classification" in target;
};

const itemKey = (item: MaskData) => {
  return getMaskDataIdentifier(item);
};

export function DatabaseCatalogPanel({ database }: { database: Database }) {
  const { t } = useTranslation();
  const policyStore = usePolicyV1Store();
  const databaseCatalogStore = useDatabaseCatalogV1Store();
  const settingStore = useSettingV1Store();

  const databaseCatalog = useDatabaseCatalog(database.name, false);
  const catalog = useVueState(() => databaseCatalog.value);
  const project = getDatabaseProject(database);
  const instance = getInstanceResource(database);
  const isMaskingForNoSQL = instanceV1MaskingForNoSQL(instance);
  const hasUpdateCatalogPermission = hasProjectPermissionV2(
    project,
    "bb.databaseCatalogs.update"
  );
  const hasGrantAccessPermission = GRANT_ACCESS_PERMISSIONS.every(
    (permission) => hasProjectPermissionV2(project, permission)
  );

  const semanticTypeList = useVueState(() => {
    const setting = settingStore.getSettingByName(
      Setting_SettingName.SEMANTIC_TYPES
    );
    return setting?.value?.value.case === "semanticType"
      ? ((setting.value.value.value.types ??
          []) as SemanticTypeSetting_SemanticType[])
      : [];
  });
  const classificationConfig = useVueState(() =>
    settingStore.getProjectClassification(
      project.dataClassificationConfigId ?? ""
    )
  );
  const hasSensitiveDataFeature = useVueState(
    () => featureToRef(PlanFeature.FEATURE_DATA_MASKING, instance).value
  );

  const [searchText, setSearchText] = useState("");
  const [checkedColumnList, setCheckedColumnList] = useState<MaskData[]>([]);
  const [showFeatureDialog, setShowFeatureDialog] = useState(false);
  const [showGrantAccessDialog, setShowGrantAccessDialog] = useState(false);
  const [pendingDeleteItem, setPendingDeleteItem] = useState<MaskData | null>(
    null
  );

  useEffect(() => {
    void settingStore.getOrFetchSettingByName(
      Setting_SettingName.SEMANTIC_TYPES,
      true
    );
    void settingStore.getOrFetchSettingByName(
      Setting_SettingName.DATA_CLASSIFICATION,
      true
    );
  }, [settingStore]);

  useEffect(() => {
    setSearchText("");
    setCheckedColumnList([]);
    setShowFeatureDialog(false);
    setShowGrantAccessDialog(false);
    setPendingDeleteItem(null);
  }, [database.name]);

  const semanticTypeOptions = useMemo(
    () =>
      semanticTypeList.map((semanticType) => ({
        label: semanticType.title || semanticType.id,
        value: semanticType.id,
      })),
    [semanticTypeList]
  );
  const classificationOptions = useMemo(
    () =>
      Object.values(classificationConfig?.classification ?? {}).map(
        (classification) => ({
          label: classification.title || classification.id,
          value: classification.id,
        })
      ),
    [classificationConfig]
  );
  const columnList = useMemo(
    () => flattenSensitiveColumnList(catalog),
    [catalog]
  );
  const filteredColumnList = useMemo(() => {
    const keyword = searchText.trim().toLowerCase();
    if (!keyword) {
      return columnList;
    }
    return columnList.filter((item) => {
      return (
        item.schema.toLowerCase().includes(keyword) ||
        item.table.toLowerCase().includes(keyword) ||
        item.column.toLowerCase().includes(keyword)
      );
    });
  }, [columnList, searchText]);
  const grantAccessColumnList = useMemo(
    () =>
      checkedColumnList.map((maskData) => ({
        database,
        maskData,
      })),
    [checkedColumnList, database]
  );
  const openGrantAccessDialog =
    showGrantAccessDialog && checkedColumnList.length > 0;

  const removeMaskingExceptions = async (sensitiveColumn: MaskData) => {
    const policy = await policyStore.getOrFetchPolicyByParentAndType({
      parentPath: database.project,
      policyType: PolicyType.MASKING_EXEMPTION,
    });
    if (!policy) {
      return;
    }

    const exemptions = (
      policy.policy?.case === "maskingExemptionPolicy"
        ? policy.policy.value.exemptions
        : []
    ).filter(
      (exception) =>
        !isCurrentColumnException(exception, {
          database,
          maskData: sensitiveColumn,
        })
    );

    policy.policy = {
      case: "maskingExemptionPolicy",
      value: create(MaskingExemptionPolicySchema, {
        exemptions,
      }),
    };

    await policyStore.upsertPolicy({
      parentPath: database.project,
      policy,
    });
  };

  const handleDelete = async (item: MaskData) => {
    setCheckedColumnList((current) =>
      current.filter(
        (selectedColumn) => itemKey(selectedColumn) !== itemKey(item)
      )
    );
    if (hasSemanticType(item.target)) {
      item.target.semanticType = "";
      item.semanticTypeId = "";
    }
    if (hasClassificationType(item.target)) {
      item.target.classification = "";
      item.classificationId = "";
    }
    await databaseCatalogStore.updateDatabaseCatalog(catalog);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.removed"),
    });
    await removeMaskingExceptions(item);
  };

  const handleSemanticTypeChange = async (
    item: MaskData,
    semanticTypeId: string
  ) => {
    if (!hasSemanticType(item.target)) {
      return;
    }
    item.target.semanticType = semanticTypeId;
    item.semanticTypeId = semanticTypeId;
    await databaseCatalogStore.updateDatabaseCatalog(catalog);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  };

  const handleClassificationChange = async (
    item: MaskData,
    classificationId: string
  ) => {
    if (!hasClassificationType(item.target)) {
      return;
    }
    item.target.classification = classificationId;
    item.classificationId = classificationId;
    await databaseCatalogStore.updateDatabaseCatalog(catalog);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  };

  const handleGrantAccessClick = () => {
    if (!hasSensitiveDataFeature) {
      setShowFeatureDialog(true);
      return;
    }
    setShowGrantAccessDialog(true);
  };

  const closeGrantAccessDialog = () => {
    setShowGrantAccessDialog(false);
    setCheckedColumnList([]);
  };

  const handleDeleteConfirmed = async () => {
    if (!pendingDeleteItem) {
      return;
    }
    const item = pendingDeleteItem;
    setPendingDeleteItem(null);
    await handleDelete(item);
  };

  return (
    <div className="flex flex-col gap-y-4">
      <FeatureAttention
        feature={PlanFeature.FEATURE_DATA_MASKING}
        instance={instance}
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <Input
          value={searchText}
          onChange={(event) => setSearchText(event.target.value)}
          placeholder={t("common.search")}
          className="w-full max-w-sm"
        />

        {!isMaskingForNoSQL && (
          <PermissionGuard
            permissions={GRANT_ACCESS_PERMISSIONS}
            project={project}
          >
            {({ disabled }) => (
              <Button
                type="button"
                className="w-full sm:w-auto"
                disabled={
                  disabled ||
                  !hasGrantAccessPermission ||
                  checkedColumnList.length === 0
                }
                onClick={handleGrantAccessClick}
              >
                {hasSensitiveDataFeature ? (
                  <ShieldCheck className="w-4 h-4" />
                ) : (
                  <FeatureBadge
                    feature={PlanFeature.FEATURE_DATA_MASKING}
                    instance={instance}
                    clickable={false}
                    className="text-white"
                  />
                )}
                {t("settings.sensitive-data.grant-access")}
              </Button>
            )}
          </PermissionGuard>
        )}
      </div>

      <SensitiveColumnTable
        database={database}
        columnList={filteredColumnList}
        checkedColumnList={checkedColumnList}
        showSelection={!isMaskingForNoSQL}
        canEdit={hasUpdateCatalogPermission && hasSensitiveDataFeature}
        showOperation={hasUpdateCatalogPermission && hasSensitiveDataFeature}
        semanticTypeOptions={semanticTypeOptions}
        classificationOptions={classificationOptions}
        onCheckedColumnListChange={(columnList) =>
          setCheckedColumnList(columnList)
        }
        onSemanticTypeChange={handleSemanticTypeChange}
        onClassificationChange={handleClassificationChange}
        onDelete={(item) => setPendingDeleteItem(item)}
      />

      <Dialog open={showFeatureDialog} onOpenChange={setShowFeatureDialog}>
        <DialogContent className="p-6">
          <DialogTitle>{t("common.warning")}</DialogTitle>
          <div className="mt-3">
            <FeatureAttention
              feature={PlanFeature.FEATURE_DATA_MASKING}
              instance={instance}
            />
          </div>
          <div className="mt-6 flex justify-end gap-x-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => setShowFeatureDialog(false)}
            >
              {t("common.cancel")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <GrantAccessDialog
        open={openGrantAccessDialog}
        projectName={database.project}
        columnList={grantAccessColumnList}
        instance={instance}
        onDismiss={closeGrantAccessDialog}
      />

      <AlertDialog
        open={pendingDeleteItem !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingDeleteItem(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogTitle>{t("common.warning")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("settings.sensitive-data.remove-sensitive-column-tips")}
          </AlertDialogDescription>
          <div className="mt-4 flex flex-col gap-2 text-sm text-main">
            {pendingDeleteItem ? (
              <div className="rounded-xs border border-control-border px-3 py-2">
                {pendingDeleteItem.schema
                  ? `${pendingDeleteItem.schema}.${pendingDeleteItem.table}`
                  : pendingDeleteItem.table}
                {pendingDeleteItem.column ? `.${pendingDeleteItem.column}` : ""}
              </div>
            ) : null}
          </div>
          <AlertDialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setPendingDeleteItem(null)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={() => void handleDeleteConfirmed()}
            >
              {t("common.delete")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
