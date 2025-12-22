<template>
  <div
    v-if="queriedDatabaseNames.length > 1"
    class="w-full flex flex-row justify-start items-center p-2 pb-0 gap-2 shrink-0"
  >
    <NTooltip v-if="showEmptySwitch">
      <template #trigger>
        <NButton
          tertiary
          size="small"
          :type="showEmpty ? 'primary' : 'default'"
          style="--n-padding: 6px; margin-bottom: 0.5rem"
          @click="showEmpty = !showEmpty"
        >
          <EyeIcon v-if="showEmpty" class="w-4 h-4" />
          <EyeOffIcon v-else class="w-4 h-4" />
        </NButton>
      </template>
      <template #default>
        {{ $t("sql-editor.batch-query.show-or-hide-empty-query-results") }}
      </template>
    </NTooltip>

    <DataExportButton
      tertiary
      size="small"
      :support-formats="[
        ExportFormat.CSV,
        ExportFormat.JSON,
        ExportFormat.SQL,
        ExportFormat.XLSX,
      ]"
      style="margin-bottom: 0.5rem"
      :view-mode="'DRAWER'"
      :text="$t('sql-editor.batch-export.self')"
      :tooltip="$t('sql-editor.batch-export.tooltip', { max: MAX_EXPORT })"
      :validate="validateExport"
      :support-password="true"
      :maximum-export-count="effectiveQueryDataPolicy.maximumResultRows"
      @export="handleExportBtnClick"
    >
      <template #form>
        <div class="w-full mb-6 flex flex-col gap-y-2">
          <div>
            <p class="textlabel">
              {{ $t("database.select") }}
              <RequiredStar />
            </p>
            <span class="textinfolabel">
              {{ $t("sql-editor.batch-export.tooltip", { max: MAX_EXPORT }) }}
            </span>
          </div>
          <DatabaseV1Table
            :schemaless="true"
            :mode="'PROJECT_SHORT'"
            :database-list="databaseList"
            :show-selection="true"
            :pagination="{
              defaultPageSize: MAX_EXPORT,
              disabled: false,
            }"
            v-model:selected-database-names="selectedDatabaseNameList"
          />
        </div>
      </template>
    </DataExportButton>

    <NScrollbar x-scrollable class="pb-2">
      <div class="flex flex-row justify-start items-center gap-2">
        <NButton
          v-for="(item, index) in filteredItems"
          :key="item.database.name"
          secondary
          strong
          size="small"
          :type="'default'"
          :style="{
            ...getBackgroundColorRgb(item.database),
            borderTop: selectedDatabase === item.database ? '3px solid' : '',
          }"
          @contextmenu.stop.prevent="
            contextMenuRef?.show(index, $event)
          "
          @click="$emit('update:selected-database', item.database)"
        >
          <RichDatabaseName :database="item.database" />
          <CircleAlertIcon
            v-if="isDatabaseQueryFailed(item)"
            class="ml-1 text-red-600 w-4 h-auto"
          />
          <span
            v-if="isEmptyQueryItem(item)"
            class="text-control-placeholder italic ml-1"
          >
            ({{ $t("common.empty") }})
          </span>
          <XIcon
            class="ml-1 text-gray-400 w-4 h-auto hover:text-gray-600"
            @click.stop="handleCloseSingleResultView(item)"
          />
        </NButton>
      </div>
    </NScrollbar>
    <ContextMenu ref="contextMenuRef" />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { head } from "lodash-es";
import { CircleAlertIcon, EyeIcon, EyeOffIcon, XIcon } from "lucide-vue-next";
import { NButton, NScrollbar, NTooltip } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, ref, watch } from "vue";
import type {
  DownloadContent,
  ExportOption,
} from "@/components/DataExportButton.vue";
import DataExportButton from "@/components/DataExportButton.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { RichDatabaseName } from "@/components/v2";
import { DatabaseV1Table } from "@/components/v2/Model/DatabaseV1Table";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useDatabaseV1Store,
  usePolicyV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLStore,
} from "@/store";
import type { ComposedDatabase, SQLEditorDatabaseQueryContext } from "@/types";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { ExportRequestSchema } from "@/types/proto-es/v1/sql_service_pb";
import { hexToRgb } from "@/utils";
import ContextMenu from "./ContextMenu.vue";
import { provideResultTabListContext } from "./context";

const MAX_EXPORT = 20;

type BatchQueryItem = {
  database: ComposedDatabase;
  context: SQLEditorDatabaseQueryContext | undefined;
};

const props = defineProps<{
  selectedDatabase: ComposedDatabase | undefined;
}>();

const emit = defineEmits<{
  (event: "update:selected-database", db: ComposedDatabase | undefined): void;
}>();

const tabStore = useSQLEditorTabStore();
const { currentTab: tab } = storeToRefs(tabStore);
const databaseStore = useDatabaseV1Store();
const sqlStore = useSQLStore();
const showEmpty = ref<boolean>(true);
const policyStore = usePolicyV1Store();
const { project } = storeToRefs(useSQLEditorStore());
const contextMenu = provideResultTabListContext();
const contextMenuRef = ref<InstanceType<typeof ContextMenu>>();

const effectiveQueryDataPolicy = computed(() => {
  return policyStore.getEffectiveQueryDataPolicyForProject(project.value);
});

const selectedDatabaseNameList = ref<string[]>([]);

const queriedDatabaseNames = computed(() =>
  Array.from(tab.value?.databaseQueryContexts?.keys() || [])
);

const databaseList = computed(() => {
  return queriedDatabaseNames.value.map((name) =>
    databaseStore.getDatabaseByName(name)
  );
});

const validateExport = () => {
  return (
    selectedDatabaseNameList.value.length > 0 &&
    selectedDatabaseNameList.value.length <= MAX_EXPORT
  );
};

const items = computed(() => {
  return queriedDatabaseNames.value.map<BatchQueryItem>((name) => {
    const database = databaseStore.getDatabaseByName(name);
    const context = head(tab.value?.databaseQueryContexts?.get(name));
    return { database, context };
  });
});

const isEmptyQueryItem = (item: BatchQueryItem) => {
  if (!item.context) {
    return true;
  }
  if (item.context.resultSet?.error) {
    // Failed queries have empty result sets, but should not be recognized
    // as empty result sets.
    return false;
  }
  if (item.context.status !== "DONE") {
    return false;
  }
  return item.context.resultSet?.results.every(
    (result) => result.rows.length === 0
  );
};

const showEmptySwitch = computed(() => {
  if (items.value.length <= 1) {
    return false;
  }
  return items.value.some((item) => isEmptyQueryItem(item));
});

const filteredItems = computed(() => {
  if (showEmpty.value || !showEmptySwitch.value) {
    return items.value;
  }

  return items.value.filter((item) => !isEmptyQueryItem(item));
});

const isDatabaseQueryFailed = (item: BatchQueryItem) => {
  // If there is any error in the result set, we consider the query failed.
  return (
    item.context?.resultSet?.error ||
    item.context?.resultSet?.results.find((result) => result.error)
  );
};

const handleCloseSingleResultView = (item: BatchQueryItem) => {
  const contexts = tab.value?.databaseQueryContexts?.get(item.database.name);
  if (!contexts) {
    return;
  }
  for (const context of contexts) {
    context.abortController?.abort();
  }
  tab.value?.databaseQueryContexts?.delete(item.database.name);
};

// Auto select a proper database when the databases are ready.
watch(
  () => filteredItems.value,
  (items) => {
    const curr = props.selectedDatabase;
    if (!curr || !items.find((item) => item.database === curr)) {
      emit("update:selected-database", head(items)?.database);
    }
  },
  {
    immediate: true,
  }
);

const getBackgroundColorRgb = (database: ComposedDatabase) => {
  const color = hexToRgb(
    database.effectiveEnvironmentEntity.color || "#4f46e5"
  ).join(", ");
  return {
    backgroundColor: `rgba(${color}, 0.1)`,
    borderTopColor: `rgb(${color})`,
    color: `rgb(${color})`,
    borderTop: "3px solid",
  };
};

const handleExportBtnClick = async ({
  options,
  resolve,
}: {
  options: ExportOption;
  reject: (reason?: unknown) => void;
  resolve: (content: DownloadContent[]) => void;
}) => {
  const contents: DownloadContent[] = [];
  for (const databaseName of selectedDatabaseNameList.value) {
    const database = databaseStore.getDatabaseByName(databaseName);
    const context = head(tab.value?.databaseQueryContexts?.get(databaseName));
    if (!context) {
      continue;
    }
    try {
      const content = await sqlStore.exportData(
        create(ExportRequestSchema, {
          name: databaseName,
          dataSourceId: context.params.connection.dataSourceId ?? "",
          format: options.format,
          statement: context.params.statement,
          limit: options.limit,
          admin: tabStore.currentTab?.mode === "ADMIN",
          password: options.password,
          schema: context.params.connection.schema,
        })
      );

      contents.push({
        content,
        // the download file is always zip file.
        filename: `${database.databaseName}.${dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss")}.zip`,
      });
    } catch (e) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("sql-editor.batch-export.failed-for-db", {
          db: databaseName,
        }),
        description: String(e),
      });
    }
  }

  resolve(contents);
  selectedDatabaseNameList.value = [];
};

useEmitteryEventListener(
  contextMenu.events,
  "close-tab",
  async ({ index, action }) => {
    const max = items.value.length - 1;
    switch (action) {
      case "CLOSE": {
        handleCloseSingleResultView(items.value[index]);
        return;
      }
      case "CLOSE_OTHERS": {
        for (let i = max; i > index; i--) {
          handleCloseSingleResultView(items.value[i]);
        }
        for (let i = index - 1; i >= 0; i--) {
          handleCloseSingleResultView(items.value[i]);
        }
        return;
      }
      case "CLOSE_TO_THE_RIGHT": {
        for (let i = max; i > index; i--) {
          handleCloseSingleResultView(items.value[i]);
        }
        return;
      }
      case "CLOSE_ALL": {
        for (let i = max; i >= 0; i--) {
          handleCloseSingleResultView(items.value[i]);
        }
        return;
      }
    }
  }
);
</script>
