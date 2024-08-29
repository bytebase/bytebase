<template>
  <div
    class="text-md font-normal flex flex-col gap-2"
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
    <template v-else>
      <div class="flex items-start gap-2">
        <div class="shrink-0 flex items-center h-6">
          <CircleAlertIcon class="w-6 h-6 text-error" />
        </div>
        <span>{{ error }}</span>
      </div>
    </template>
    <div>
      <slot name="suffix" />
    </div>
    <div v-if="showRunAnywayButton">
      <NButton size="small" type="primary" @click="runAnyway">
        {{ $t("sql-editor.run-anyway") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { CircleAlertIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { Status } from "nice-grpc-common";
import { computed } from "vue";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { useAppFeature } from "@/store";
import type { SQLEditorQueryParams, SQLResultSetV1 } from "@/types";
import { Advice_Status } from "@/types/proto/v1/sql_service";
import { useSQLResultViewContext } from "../context";
import AdviceItem from "./AdviceItem.vue";

const props = defineProps<{
  error: string | undefined;
  executeParams?: SQLEditorQueryParams;
  resultSet?: SQLResultSetV1;
}>();

const sqlCheckStyle = useAppFeature("bb.feature.sql-editor.sql-check-style");
const { dark } = useSQLResultViewContext();
const { execute } = useExecuteSQL();

const showRunAnywayButton = computed(() => {
  const { executeParams, resultSet } = props;
  if (!resultSet) return false;
  if (!executeParams) return false;
  if (sqlCheckStyle.value !== "PREFLIGHT") return false;
  if (resultSet.status === Status.PERMISSION_DENIED) return false;
  if (resultSet.error.includes("resource not found")) return false;
  if (
    resultSet.advices.some((advice) => advice.status === Advice_Status.ERROR)
  ) {
    return false;
  }
  return true;
});

const runAnyway = () => {
  const params = props.executeParams;
  if (!params) return;
  execute({
    ...params,
    skipCheck: true,
  });
};
</script>
