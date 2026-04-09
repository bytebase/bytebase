import { create } from "@bufbuild/protobuf";
import { ShieldCheck } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type {
  MaskData,
  MaskDataTarget,
} from "@/components/SensitiveData/types";
import { isCurrentColumnException } from "@/components/SensitiveData/utils";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import {
  featureToRef,
  pushNotification,
  useDatabaseCatalog,
  useDatabaseCatalogV1Store,
  usePolicyV1Store,
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
      const result = [];
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

  const sensitiveList: MaskData[] = [];

  for (const schema of catalog.schemas) {
    for (const table of schema.tables) {
      if (table.kind?.case === "columns") {
        for (const column of table.kind.value.columns ?? []) {
          if (!column.semanticType && !column.classification) {
            continue;
          }
          sensitiveList.push({
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
        sensitiveList.push(
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
        sensitiveList.push({
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

  return sensitiveList;
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

export function DatabaseCatalogPanel({ database }: { database: Database }) {
  const { t } = useTranslation();
  const policyStore = usePolicyV1Store();
  const databaseCatalogStore = useDatabaseCatalogV1Store();

  const databaseCatalog = useDatabaseCatalog(database.name, false);
  const catalog = useVueState(() => databaseCatalog.value);
  const hasSensitiveDataFeature = useVueState(
    () => featureToRef(PlanFeature.FEATURE_DATA_MASKING).value
  );

  const [searchText, setSearchText] = useState("");
  const [checkedColumnList, setCheckedColumnList] = useState<MaskData[]>([]);
  const [showFeatureDialog, setShowFeatureDialog] = useState(false);
  const [showGrantAccessDrawer, setShowGrantAccessDrawer] = useState(false);

  const instance = getInstanceResource(database);
  const isMaskingForNoSQL = instanceV1MaskingForNoSQL(instance);
  const project = getDatabaseProject(database);
  const hasUpdateCatalogPermission = hasProjectPermissionV2(
    project,
    "bb.databaseCatalogs.update"
  );
  const hasGrantAccessPermission = GRANT_ACCESS_PERMISSIONS.every(
    (permission) => hasProjectPermissionV2(project, permission)
  );

  const columnList = useMemo(
    () => flattenSensitiveColumnList(catalog),
    [catalog]
  );
  const normalizedSearchText = searchText.trim().toLowerCase();
  const filteredColumnList = useMemo(() => {
    if (!normalizedSearchText) {
      return columnList;
    }
    return columnList.filter((item) => {
      return (
        item.schema.toLowerCase().includes(normalizedSearchText) ||
        item.table.toLowerCase().includes(normalizedSearchText) ||
        item.column.toLowerCase().includes(normalizedSearchText)
      );
    });
  }, [columnList, normalizedSearchText]);

  const openGrantAccessDrawer =
    showGrantAccessDrawer && checkedColumnList.length > 0;

  useEffect(() => {
    setSearchText("");
    setCheckedColumnList([]);
    setShowFeatureDialog(false);
    setShowGrantAccessDrawer(false);
  }, [database.name]);

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
    if (hasSemanticType(item.target)) {
      item.target.semanticType = "";
    }
    if (hasClassificationType(item.target)) {
      item.target.classification = "";
    }
    if (databaseCatalog.value) {
      await databaseCatalogStore.updateDatabaseCatalog(databaseCatalog.value);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.removed"),
      });
    }
    await removeMaskingExceptions(item);
  };

  const handleGrantAccessClick = () => {
    if (!hasSensitiveDataFeature) {
      setShowFeatureDialog(true);
      return;
    }
    setShowGrantAccessDrawer(true);
  };

  return (
    <div className="w-full flex flex-col gap-y-4">
      <FeatureAttention
        feature={PlanFeature.FEATURE_DATA_MASKING}
        instance={instance}
      />

      <div className="flex flex-col gap-y-4 gap-x-2 lg:flex-row justify-between items-end lg:items-center">
        <Input
          value={searchText}
          onChange={(event) => setSearchText(event.target.value)}
          placeholder={t("common.search")}
          className="w-full"
        />

        {!isMaskingForNoSQL && (
          <PermissionGuard
            permissions={GRANT_ACCESS_PERMISSIONS}
            project={project}
          >
            <Button
              onClick={handleGrantAccessClick}
              disabled={
                !hasGrantAccessPermission || checkedColumnList.length === 0
              }
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
          </PermissionGuard>
        )}
      </div>

      <SensitiveColumnTable
        rowSelectable={!isMaskingForNoSQL}
        showOperation={hasUpdateCatalogPermission && hasSensitiveDataFeature}
        columnList={filteredColumnList}
        checkedColumnList={checkedColumnList}
        onCheckedColumnListChange={setCheckedColumnList}
        onDelete={handleDelete}
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
              variant="outline"
              onClick={() => setShowFeatureDialog(false)}
            >
              {t("common.cancel")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {openGrantAccessDrawer && (
        <GrantAccessDialog
          open={openGrantAccessDrawer}
          columnList={checkedColumnList.map((maskData) => ({
            database,
            maskData,
          }))}
          projectName={database.project}
          onDismiss={() => {
            setShowGrantAccessDrawer(false);
            setCheckedColumnList([]);
          }}
        />
      )}
    </div>
  );
}
