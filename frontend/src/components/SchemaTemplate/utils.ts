import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { computed, reactive } from "vue";
import { useSettingV1Store } from "@/store";
import { unknownDatabase, type ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DatabaseCatalogSchema,
  SchemaCatalogSchema,
  type TableCatalog,
  TableCatalogSchema,
  TableCatalog_ColumnsSchema,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import {
  DatabaseMetadataSchema,
  SchemaMetadataSchema,
  type TableMetadata,
  TableMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type {
  SchemaTemplateSetting_TableTemplate,
  DataClassificationSetting_DataClassificationConfig,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  SchemaTemplateSetting_TableTemplateSchema,
  Setting_SettingName,
} from "@/types/proto-es/v1/setting_service_pb";

export const engineList = [Engine.MYSQL, Engine.POSTGRES];

// Create empty schema template table template
export const createEmptyTableTemplate =
  (): SchemaTemplateSetting_TableTemplate => {
    return create(SchemaTemplateSetting_TableTemplateSchema, {
      id: "",
      category: "",
      engine: Engine.ENGINE_UNSPECIFIED,
      table: create(TableMetadataSchema, {}),
      catalog: create(TableCatalogSchema, {
        name: "",
        kind: {
          case: "columns",
          value: create(TableCatalog_ColumnsSchema, {}),
        },
      }),
    });
  };

export const categoryList = computed(() => {
  const settingStore = useSettingV1Store();
  const setting = settingStore.getSettingByName(
    Setting_SettingName.SCHEMA_TEMPLATE
  );

  if (
    !setting ||
    !setting.value ||
    setting.value.value.case !== "schemaTemplateSettingValue"
  ) {
    return [];
  }

  const schemaTemplateSetting = setting.value.value.value;
  const fieldTemplateList = schemaTemplateSetting.fieldTemplates ?? [];
  const tableTemplateList = schemaTemplateSetting.tableTemplates ?? [];
  const resp = [];

  for (const category of new Set([
    ...fieldTemplateList.map((template) => template.category),
    ...tableTemplateList.map((template) => template.category),
  ])) {
    if (!category) {
      continue;
    }
    resp.push(category);
  }
  return resp;
});

export const classificationConfig = computed(
  (): DataClassificationSetting_DataClassificationConfig | undefined => {
    const settingStore = useSettingV1Store();
    const setting = settingStore.getSettingByName(
      Setting_SettingName.DATA_CLASSIFICATION
    );

    if (
      !setting ||
      !setting.value ||
      setting.value.value.case !== "dataClassificationSettingValue"
    ) {
      return undefined;
    }

    const configs = setting.value.value.value.configs;
    return configs.length > 0 ? configs[0] : undefined;
  }
);

export const mockMetadataFromTableTemplate = (
  template: SchemaTemplateSetting_TableTemplate
) => {
  const db = {
    ...unknownDatabase(),
  };
  const tableMetadata =
    cloneDeep(template.table) ?? create(TableMetadataSchema, {});
  const schemaMetadata = create(SchemaMetadataSchema, {
    name: "",
    tables: [tableMetadata],
  });

  const tableCatalog =
    cloneDeep(template.catalog) ??
    create(TableCatalogSchema, {
      name: "",
      kind: {
        case: "columns",
        value: create(TableCatalog_ColumnsSchema, {}),
      },
    });
  const databaseMetadata = create(DatabaseMetadataSchema, {
    name: db.name,
    schemas: [
      create(SchemaMetadataSchema, {
        name: "",
        tables: [cloneDeep(template.table) ?? create(TableMetadataSchema, {})],
      }),
    ],
  });
  const schemaCatalog = create(SchemaCatalogSchema, {
    name: "",
    tables: [tableCatalog],
  });
  const databaseCatalog = create(DatabaseCatalogSchema, {
    name: db.name,
    schemas: [schemaCatalog],
  });
  return reactive({
    db,
    databaseMetadata,
    databaseCatalog,
    schemaCatalog,
    schemaMetadata,
    tableCatalog,
    tableMetadata,
    id: template.id,
    category: template.category,
    engine: template.engine,
  });
};

export const rebuildTableTemplateFromMetadata = (params: {
  db: ComposedDatabase;
  tableMetadata: TableMetadata;
  tableCatalog: TableCatalog;
  id: string;
  category: string;
  engine: Engine;
}) => {
  const { tableMetadata, tableCatalog, id, category, engine } = params;

  return create(SchemaTemplateSetting_TableTemplateSchema, {
    table: tableMetadata,
    catalog: tableCatalog,
    id,
    category,
    engine,
  });
};
