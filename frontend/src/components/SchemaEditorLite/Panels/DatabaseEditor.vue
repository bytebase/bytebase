<template>
  <div class="flex flex-col w-full h-full overflow-y-hidden" v-bind="$attrs">
    <div class="py-2 w-full flex flex-row justify-between items-center">
      <div>
        <div
          v-if="state.selectedSubTab === 'table-list'"
          class="w-full flex justify-between items-center gap-x-2"
        >
          <div class="flex flex-row justify-start items-center gap-x-2">
            <div
              v-if="shouldShowSchemaSelector"
              class="pl-1 flex-row justify-start items-center text-sm gap-x-2 overflow-auto hidden xl:flex"
            >
              <span class="shrink-0">Schema:</span>
              <NSelect
                :value="selectedSchemaName"
                :options="schemaSelectorOptionList"
                class="min-w-32"
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
          class="flex flex-row justify-end items-center bg-gray-100 p-1 rounded-sm whitespace-nowrap"
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
</template>

<script lang="ts" setup>
import { head, sumBy } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton, NSelect, NTooltip } from "naive-ui";
import { computed, nextTick, reactive, watch } from "vue";
import SchemaDiagram, { SchemaDiagramIcon } from "@/components/SchemaDiagram";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";
import TableNameModal from "../Modals/TableNameModal.vue";
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

type SubTabType = "table-list" | "schema-diagram";

interface LocalState {
  selectedSubTab: SubTabType;
  showFeatureModal: boolean;
  tableNameModalContext?: {
    schema: SchemaMetadata;
  };
  activeTableId?: string;
}

const context = useSchemaEditorContext();
const { readonly, selectionEnabled, addTab, getSchemaStatus } = context;
const state = reactive<LocalState>({
  selectedSubTab: "table-list",
  showFeatureModal: false,
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
  for (const schema of props.database.schemas) {
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
</script>
