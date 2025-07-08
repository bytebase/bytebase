<template>
  <div
    class="w-full text-md font-normal flex flex-col gap-2 text-sm"
    :class="[dark ? 'text-matrix-green-hover' : 'text-control-light']"
  >
    <template v-if="resultSet && resultSet.advices.length > 0">
      <AdviceItem
        v-for="(advice, i) in resultSet.advices"
        :key="i"
        :advice="advice"
        :execute-params="executeParams"
      />
    </template>
    <BBAttention v-else class="w-full" type="error">
      {{ error }}
    </BBAttention>
    <div v-if="$slots.suffix">
      <slot name="suffix" />
    </div>
    <PostgresError v-if="resultSet" :result-set="resultSet" />
    <div v-if="showRunAnywayButton">
      <NButton size="small" type="warning" @click="runAnyway">
        {{ $t("sql-editor.run-anyway") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Code } from "@connectrpc/connect";
import { NButton } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { BBAttention } from "@/bbkit";
import { useAppFeature, useSQLEditorTabStore } from "@/store";
import type { SQLEditorQueryParams, SQLResultSetV1 } from "@/types";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";
import { Advice_Status } from "@/types/proto-es/v1/sql_service_pb";
import { useSQLResultViewContext } from "../context";
import AdviceItem from "./AdviceItem.vue";
import PostgresError from "./PostgresError.vue";

const props = defineProps<{
  error: string | undefined;
  executeParams?: SQLEditorQueryParams;
  resultSet?: SQLResultSetV1;
}>();

const emit = defineEmits<{
  (event: "execute", params: SQLEditorQueryParams): void;
}>();

const sqlCheckStyle = useAppFeature("bb.feature.sql-editor.sql-check-style");
const { currentTab: tab } = storeToRefs(useSQLEditorTabStore());
const { dark } = useSQLResultViewContext();
const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");

const showRunAnywayButton = computed(() => {
  if (!tab.value) return false;
  if (tab.value?.mode === "ADMIN") return false;
  const { executeParams, resultSet } = props;
  if (!resultSet) return false;
  if (!executeParams) return false;
  if (sqlCheckStyle.value !== "PREFLIGHT") return false;
  if (databaseChangeMode.value !== DatabaseChangeMode.EDITOR) return false;
  if (resultSet.status === Code.PermissionDenied) return false;
  if (resultSet.error.includes("resource not found")) return false;
  return resultSet.advices.some(
    (advice) => advice.status === Advice_Status.WARNING
  );
});

const runAnyway = () => {
  const params = props.executeParams;
  if (!params) return;
  emit("execute", {
    ...params,
    skipCheck: true,
  });
};
</script>
