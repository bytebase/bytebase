<template>
  <div
    class="text-md font-normal flex flex-col gap-2"
    :class="[dark ? 'text-matrix-green-hover' : 'text-control-light']"
  >
    <div class="flex items-center gap-3">
      <span>{{ error }}</span>
      <slot name="suffix" />
    </div>
    <div v-if="viewMode === 'RICH'">
      <NButton
        v-if="showRunAnywayButton"
        size="small"
        type="primary"
        @click="runAnyway"
      >
        {{ $t("sql-editor.run-anyway") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { useAppFeature } from "@/store";
import type { SQLEditorQueryParams, SQLResultSetV1 } from "@/types";
import { Advice_Status } from "@/types/proto/v1/sql_service";
import { useSQLResultViewContext } from "./context";

const props = defineProps<{
  error: string | undefined;
  executeParams?: SQLEditorQueryParams;
  resultSet?: SQLResultSetV1;
}>();

const sqlCheckStyle = useAppFeature("bb.feature.sql-editor.sql-check-style");
const { dark } = useSQLResultViewContext();
const { execute } = useExecuteSQL();

const viewMode = computed((): "SIMPLE" | "RICH" => {
  if (sqlCheckStyle.value === "PREFLIGHT") {
    if (!props.executeParams?.skipCheck) {
      return "RICH";
    }
  }
  return "SIMPLE";
});
const showRunAnywayButton = computed(() => {
  const { resultSet } = props;
  if (!resultSet) return false;
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
