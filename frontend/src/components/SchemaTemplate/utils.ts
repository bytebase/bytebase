import { cloneDeep } from "lodash-es";
import { computed, reactive } from "vue";
import { useSettingV1Store } from "@/store";
import { unknownDatabase, type ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  DatabaseMetadata,
  SchemaConfig,
  SchemaMetadata,
  TableConfig,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { SchemaTemplateSetting_TableTemplate } from "@/types/proto/v1/setting_service";

export const engineList = [Engine.MYSQL, Engine.POSTGRES];

export const categoryList = computed(() => {
  const settingStore = useSettingV1Store();
  const setting = settingStore.getSettingByName("bb.workspace.schema-template");
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

export const classificationConfig = computed(() => {
  const settingStore = useSettingV1Store();
  // TODO(ed): it's a temporary solution
  return settingStore.classification[0];
});

export const mockMetadataFromTableTemplate = (
  template: SchemaTemplateSetting_TableTemplate
) => {
  const db = {
    ...unknownDatabase(),
  };
  const table = cloneDeep(template.table) ?? TableMetadata.fromPartial({});
  const schema = SchemaMetadata.fromPartial({
    name: "",
    tables: [table],
  });
  const tableConfig =
    cloneDeep(template.config) ?? TableConfig.fromPartial({ name: "" });
  const database = DatabaseMetadata.fromPartial({
    schemas: [schema],
    schemaConfigs: [
      SchemaConfig.fromPartial({
        name: "",
        tableConfigs: [tableConfig],
      }),
    ],
  });
  return reactive({
    db,
    database,
    schema,
    table,
    id: template.id,
    category: template.category,
    engine: template.engine,
  });
};

export const rebuildTableTemplateFromMetadata = (params: {
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  id: string;
  category: string;
  engine: Engine;
}) => {
  const { database, schema, table, id, category, engine } = params;
  const tableConfig = database.schemaConfigs
    .find((sc) => sc.name === schema.name)
    ?.tableConfigs.find((tc) => tc.name === table.name);
  return SchemaTemplateSetting_TableTemplate.fromPartial({
    table,
    config: tableConfig,
    id,
    category,
    engine,
  });
};
