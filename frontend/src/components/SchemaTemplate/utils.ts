import { cloneDeep } from "lodash-es";
import { computed, reactive } from "vue";
import { useSettingV1Store } from "@/store";
import { unknownDatabase, type ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  DatabaseCatalog,
  SchemaCatalog,
  TableCatalog,
  TableCatalog_Columns,
} from "@/types/proto/v1/database_catalog_service";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import {
  SchemaTemplateSetting_TableTemplate,
  DataClassificationSetting_DataClassificationConfig,
  Setting_SettingName,
} from "@/types/proto/v1/setting_service";

export const engineList = [Engine.MYSQL, Engine.POSTGRES];

export const categoryList = computed(() => {
  const settingStore = useSettingV1Store();
  const setting = settingStore.getSettingByName(Setting_SettingName.SCHEMA_TEMPLATE);
  const fieldTemplateList =
    setting?.value?.schemaTemplateSettingValue?.fieldTemplates ?? [];
  const tableTemplateList =
    setting?.value?.schemaTemplateSettingValue?.tableTemplates ?? [];
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
    // TODO(ed): it's a temporary solution
    return settingStore.classification[0];
  }
);

export const mockMetadataFromTableTemplate = (
  template: SchemaTemplateSetting_TableTemplate
) => {
  const db = {
    ...unknownDatabase(),
  };
  const tableMetadata =
    cloneDeep(template.table) ?? TableMetadata.fromPartial({});
  const schemaMetadata = SchemaMetadata.fromPartial({
    name: "",
    tables: [tableMetadata],
  });

  const tableCatalog =
    cloneDeep(template.catalog) ??
    TableCatalog.fromPartial({
      name: "",
      columns: TableCatalog_Columns.fromPartial({}),
    });
  const databaseMetadata = DatabaseMetadata.fromPartial({
    name: db.name,
    schemas: [
      SchemaMetadata.fromPartial({
        name: "",
        tables: [cloneDeep(template.table) ?? TableMetadata.fromPartial({})],
      }),
    ],
  });
  const schemaCatalog = SchemaCatalog.fromPartial({
    name: "",
    tables: [tableCatalog],
  });
  const databaseCatalog = DatabaseCatalog.fromPartial({
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

  return SchemaTemplateSetting_TableTemplate.fromPartial({
    table: tableMetadata,
    catalog: tableCatalog,
    id,
    category,
    engine,
  });
};
