<template>
  <div class="flex flex-col w-full h-full overflow-y-hidden" v-bind="$attrs">
    <div class="py-2 w-full flex flex-row justify-between items-center">
      <div>
        <div
          v-if="state.selectedSubTab === 'table-list'"
          class="w-full flex justify-between items-center space-x-2"
        >
          <div class="flex flex-row justify-start items-center gap-x-2">
            <div
              v-if="shouldShowSchemaSelector"
              class="pl-1 flex flex-row justify-start items-center text-sm gap-x-2 overflow-auto"
            >
              <span class="shrink-0">Schema:</span>
              <NSelect
                :value="selectedSchemaName"
                :options="schemaSelectorOptionList"
                class="min-w-[8rem]"
                @update:value="$emit('update:selected-schema-name', $event)"
              />
            </div>
            <NButton
              v-if="!readonly"
              :disabled="!allowCreateTable"
              @click="handleCreateNewTable"
            >
              <template #icon>
                <PlusIcon class="w-4 h-4" />
              </template>
              {{ $t("schema-editor.actions.create-table") }}
            </NButton>
            <NButton
              v-if="!readonly"
              :disabled="!allowCreateTable"
              @click="state.showSchemaTemplateDrawer = true"
            >
              <template #icon>
                <PlusIcon class="w-4 h-4" />
              </template>
              {{ $t("schema-editor.actions.add-from-template") }}
            </NButton>
            <div
              v-if="selectionEnabled"
              class="text-sm flex flex-row items-center gap-x-2"
            >
              <span class="text-main">
                {{ $t("branch.select-tables-to-rollout") }}
              </span>
              <TableSelectionSummary
                v-if="selectedSchema"
                :db="db"
                :metadata="{
                  database,
                  schema: selectedSchema,
                }"
              />
            </div>
          </div>
        </div>
      </div>
      <div class="flex justify-end items-center">
        <div
          class="flex flex-row justify-end items-center bg-gray-100 p-1 rounded whitespace-nowrap"
        >
          <NButton
            size="small"
            :secondary="state.selectedSubTab === 'table-list'"
            :quaternary="state.selectedSubTab !== 'table-list'"
            @click="handleChangeTab('table-list')"
          >
            <heroicons-outline:queue-list class="inline w-4 h-auto mr-1" />
            {{ $t("schema-editor.tables") }}
          </NButton>
          <NTooltip :disabled="!schemaDiagramDisabled">
            <template #trigger>
              <NButton
                tag="div"
                size="small"
                :disabled="schemaDiagramDisabled"
                :secondary="state.selectedSubTab === 'schema-diagram'"
                :quaternary="state.selectedSubTab !== 'schema-diagram'"
                @click="handleChangeTab('schema-diagram')"
              >
                <SchemaDiagramIcon class="mr-1" />
                {{ $t("schema-diagram.self") }}
              </NButton>
            </template>
            <div class="whitespace-nowrap">Too many tables</div>
          </NTooltip>
        </div>
      </div>
    </div>

    <div class="flex-1 overflow-y-hidden">
      <!-- List view -->
      <template v-if="state.selectedSubTab === 'table-list'">
        <TableList
          v-if="selectedSchema"
          :db="db"
          :database="database"
          :schema="selectedSchema"
          :tables="selectedSchema.tables"
          :search-pattern="searchPattern"
        />
      </template>
      <template v-else-if="state.selectedSubTab === 'schema-diagram'">
        <!-- TODO: bring status coloring back -->
        <SchemaDiagram
          :database="db"
          :database-metadata="database"
          :editable="true"
          @edit-table="tryEditTable"
          @edit-column="tryEditColumn"
        />
      </template>
    </div>
  </div>

  <TableNameModal
    v-if="state.tableNameModalContext !== undefined"
    :database="db"
    :metadata="database"
    :schema="state.tableNameModalContext.schema"
    mode="create"
    @close="state.tableNameModalContext = undefined"
  />

  <Drawer
    :show="state.showSchemaTemplateDrawer"
    @close="state.showSchemaTemplateDrawer = false"
  >
    <DrawerContent :title="$t('schema-template.table-template.self')">
      <div class="w-[calc(100vw-36rem)] min-w-[64rem] max-w-[calc(100vw-8rem)]">
        <TableTemplates
          :engine="engine"
          :readonly="true"
          @apply="handleApplyTemplate"
        />
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { head, sumBy } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton, NSelect, NTooltip } from "naive-ui";
import { computed, nextTick, reactive, watch } from "vue";
import SchemaDiagram, { SchemaDiagramIcon } from "@/components/SchemaDiagram";
import { Drawer, DrawerContent } from "@/components/v2";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type {
  TableMetadata as NewTableMetadata,
  ColumnMetadata as NewColumnMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  TableMetadataSchema,
  ColumnMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type { SchemaTemplateSetting_TableTemplate } from "@/types/proto-es/v1/setting_service_pb";
import TableTemplates from "@/views/SchemaTemplate/TableTemplates.vue";
import TableNameModal from "../Modals/TableNameModal.vue";
import { useSchemaEditorContext } from "../context";
import TableList from "./TableList";
import TableSelectionSummary from "./TableSelectionSummary.vue";

defineOptions({
  inheritAttrs: false,
});

const props = withDefaults(
  defineProps<{
    db: ComposedDatabase;
    database: DatabaseMetadata;
    selectedSchemaName: string | undefined;
    searchPattern?: string;
  }>(),
  {
    searchPattern: "",
  }
);
const emit = defineEmits<{
  (event: "update:selected-schema-name", schema: string | undefined): void;
}>();

// Conversion function for TableMetadata at service boundaries
const convertNewTableToOld = (newTable: NewTableMetadata): TableMetadata => {
  return create(TableMetadataSchema, {
    name: newTable.name,
    columns: newTable.columns.map(convertNewColumnToOld),
    engine: newTable.engine,
    collation: newTable.collation,
    userComment: newTable.userComment,
    comment: newTable.comment,
    indexes: [], // Initialize empty, will be handled separately if needed
    partitions: [], // Initialize empty, will be handled separately if needed
    foreignKeys: [], // Initialize empty, will be handled separately if needed
  });
};

// Conversion function for ColumnMetadata at service boundaries
const convertNewColumnToOld = (
  newColumn: NewColumnMetadata
): ColumnMetadata => {
  return create(ColumnMetadataSchema, {
    name: newColumn.name,
    position: newColumn.position,
    hasDefault: newColumn.hasDefault,
    default: newColumn.default,
    onUpdate: newColumn.onUpdate,
    nullable: newColumn.nullable,
    type: newColumn.type,
    characterSet: newColumn.characterSet,
    collation: newColumn.collation,
    userComment: newColumn.userComment,
    comment: newColumn.comment,
    // classification, labels, effectiveMaskingLevel are not available in old proto types
  });
};

type SubTabType = "table-list" | "schema-diagram";

interface LocalState {
  selectedSubTab: SubTabType;
  showFeatureModal: boolean;
  showSchemaTemplateDrawer: boolean;
  tableNameModalContext?: {
    schema: SchemaMetadata;
  };
  activeTableId?: string;
}

const context = useSchemaEditorContext();
const {
  readonly,
  selectionEnabled,
  disableDiffColoring,
  events,
  addTab,
  getSchemaStatus,
  markEditStatus,
  upsertTableCatalog,
  queuePendingScrollToTable,
} = context;
const state = reactive<LocalState>({
  selectedSubTab: "table-list",
  showFeatureModal: false,
  showSchemaTemplateDrawer: false,
});
const engine = computed(() => {
  return props.db.instanceResource.engine;
});
const selectedSchema = computed(() => {
  return props.database.schemas.find(
    (schema) => schema.name === props.selectedSchemaName
  );
});
const shouldShowSchemaSelector = computed(() => {
  return engine.value === Engine.POSTGRES;
});

const allowCreateTable = computed(() => {
  const schema = selectedSchema.value;
  if (!schema) return false;
  if (engine.value === Engine.POSTGRES) {
    const status = getSchemaStatus(props.db, {
      schema,
    });

    return (
      props.database.schemas.length > 0 &&
      selectedSchema.value &&
      status !== "dropped"
    );
  }
  return true;
});

const schemaSelectorOptionList = computed(() => {
  const optionList = [];
  const schemas = disableDiffColoring.value
    ? props.database.schemas.filter((schema) => {
        const status = getSchemaStatus(props.db, {
          schema,
        });
        return status !== "dropped";
      })
    : props.database.schemas;
  for (const schema of schemas) {
    optionList.push({
      label: schema.name,
      value: schema.name,
    });
  }

  return optionList;
});

watch(
  schemaSelectorOptionList,
  (options) => {
    if (!options.find((opt) => opt.value === props.selectedSchemaName)) {
      emit("update:selected-schema-name", head(options)?.value);
    }
  },
  {
    immediate: true,
  }
);

const handleChangeTab = (tab: SubTabType) => {
  state.selectedSubTab = tab;
};
const schemaDiagramDisabled = computed(() => {
  return sumBy(props.database.schemas, (schema) => schema.tables.length) >= 200;
});
watch(
  schemaDiagramDisabled,
  (disabled) => {
    if (disabled) handleChangeTab("table-list");
  },
  { immediate: true }
);

const handleCreateNewTable = () => {
  if (selectedSchema.value) {
    state.tableNameModalContext = {
      schema: selectedSchema.value,
    };
  }
};

const tryEditTable = async (schema: SchemaMetadata, table: TableMetadata) => {
  emit("update:selected-schema-name", schema.name);
  await nextTick();
  addTab({
    type: "table",
    database: props.db,
    metadata: {
      database: props.database,
      schema,
      table,
    },
  });
};

const tryEditColumn = async (
  schema: SchemaMetadata,
  table: TableMetadata,
  column: ColumnMetadata
) => {
  if (schema && table && column) {
    await tryEditTable(schema, table);

    // TODO: scroll column into view and focus the input box
    // await nextTick();
    // const container = document.querySelector("#table-editor-container");
    // const input = container?.querySelector(
    //   `.column-${column.id} .column-${target}-input`
    // ) as HTMLInputElement | undefined;
    // if (input) {
    //   input.focus();
    //   scrollIntoView(input);
    // }
  }
};

const handleApplyTemplate = (template: SchemaTemplateSetting_TableTemplate) => {
  state.showSchemaTemplateDrawer = false;
  if (!template.table) {
    return;
  }
  if (template.engine !== engine.value) {
    return;
  }

  const table = convertNewTableToOld(template.table);
  const schema = selectedSchema.value;
  if (!schema) {
    return;
  }
  schema.tables.push(table);
  const metadataForTable = () => {
    return {
      database: props.database,
      schema,
      table,
    };
  };
  const { db } = props;
  if (template.catalog) {
    upsertTableCatalog(
      {
        database: props.db.name,
        schema: schema.name,
        table: table.name,
      },
      (catalog) => {
        Object.assign(catalog, template.catalog);
      }
    );
  }
  markEditStatus(db, metadataForTable(), "created");
  table.columns.forEach((column) => {
    markEditStatus(db, { ...metadataForTable(), column }, "created");
  });

  addTab({
    type: "table",
    database: db,
    metadata: metadataForTable(),
  });

  queuePendingScrollToTable({
    db,
    metadata: metadataForTable(),
  });

  events.emit("rebuild-tree", {
    openFirstChild: false,
  });
};
</script>
