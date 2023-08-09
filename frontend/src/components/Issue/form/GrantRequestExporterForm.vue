<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start mb-4 space-y-4"
  >
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex items-center textlabel mb-2">
        {{ $t("common.database") }}
      </span>
      <DatabaseResourceTable
        class="w-full"
        :database-resource-list="
          selectedDatabaseResource ? [selectedDatabaseResource] : []
        "
      />
    </div>
    <div
      v-if="exportMethod === 'SQL'"
      class="w-full flex flex-col justify-start items-start"
    >
      <span class="flex items-center textlabel mb-2">SQL</span>
      <div class="w-full border rounded">
        <MonacoEditor
          class="w-full h-[300px] py-2"
          readonly
          :value="state.statement"
          :auto-focus="false"
          :language="'sql'"
          :dialect="dialect"
        />
      </div>
    </div>
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex items-center textlabel mb-2">
        {{ $t("issue.grant-request.export-rows") }}
      </span>
      <div class="flex flex-row justify-start items-start">
        {{ state.maxRowCount }}
      </div>
    </div>
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex items-center textlabel mb-2">
        {{ $t("issue.grant-request.export-format") }}
      </span>
      <div class="flex flex-row justify-start items-start">
        {{ state.exportFormat }}
      </div>
    </div>
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex items-center textlabel mb-2">
        {{ $t("issue.grant-request.expired-at") }}
      </span>
      <div>
        {{ state.expiredAt }}
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { head } from "lodash-es";
import { computed, onMounted, reactive, ref } from "vue";
import MonacoEditor from "@/components/MonacoEditor";
import { useDatabaseV1Store } from "@/store";
import {
  GrantRequestPayload,
  Issue,
  PresetRoleType,
  SQLDialect,
  UNKNOWN_ID,
  dialectOfEngineV1,
} from "@/types";
import { DatabaseResource } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { convertFromCELString } from "@/utils/issue/cel";
import { useIssueLogic } from "../logic";
import DatabaseResourceTable from "../table/DatabaseResourceTable.vue";

interface LocalState {
  databaseId?: string;
  maxRowCount: number;
  exportFormat: "CSV" | "JSON";
  statement: string;
  expiredAt: string;
}

const { issue } = useIssueLogic();
const databaseStore = useDatabaseV1Store();
const state = reactive<LocalState>({
  maxRowCount: 1000,
  exportFormat: "CSV",
  statement: "",
  expiredAt: "",
});
const selectedDatabaseResource = ref<DatabaseResource | undefined>(undefined);

const selectedDatabase = computed(() => {
  return databaseStore.getDatabaseByName(
    selectedDatabaseResource.value?.databaseName ?? String(UNKNOWN_ID)
  );
});

const exportMethod = computed(() => {
  return state.statement === "" ? "DATABASE" : "SQL";
});

const dialect = computed((): SQLDialect => {
  const db = selectedDatabase.value;
  return dialectOfEngineV1(db?.instanceEntity.engine ?? Engine.MYSQL);
});

onMounted(async () => {
  const payload = ((issue.value as Issue).payload as any)
    .grantRequest as GrantRequestPayload;
  if (payload.role !== PresetRoleType.EXPORTER) {
    throw "Only support EXPORTER role";
  }

  const conditionExpression = await convertFromCELString(
    payload.condition.expression
  );
  if (
    conditionExpression.databaseResources !== undefined &&
    conditionExpression.databaseResources.length > 0
  ) {
    const resource = head(conditionExpression.databaseResources);
    if (resource) {
      const database = await databaseStore.getOrFetchDatabaseByName(
        resource.databaseName
      );
      state.databaseId = database.uid;
      selectedDatabaseResource.value = resource;
    }
  }
  if (conditionExpression.expiredTime !== undefined) {
    state.expiredAt = dayjs(new Date(conditionExpression.expiredTime)).format(
      "LLL"
    );
  } else {
    state.expiredAt = "-";
  }
  if (conditionExpression.statement !== undefined) {
    state.statement = conditionExpression.statement;
  }
  if (conditionExpression.rowLimit !== undefined) {
    state.maxRowCount = conditionExpression.rowLimit;
  }
  if (conditionExpression.exportFormat !== undefined) {
    state.exportFormat = conditionExpression.exportFormat as "CSV" | "JSON";
  }
});
</script>
