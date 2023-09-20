<template>
  <div class="flex flex-row items-center gap-2">
    <slot name="result" :advices="advices" :is-running="isRunning">
      <div class="flex flex-col text-xs">
        <span>statement: {{ statement }}</span>
        <pre v-for="(advice, i) in advices" :key="i">{{
          Advice.toJSON(advice)
        }}</pre>
      </div>
    </slot>

    <NButton
      style="--n-padding: 0 14px 0 10px"
      :disabled="statement.trim().length === 0"
      :style="buttonStyle"
      v-bind="buttonProps"
      @click="runChecks"
    >
      <template #icon>
        <BBSpin v-if="isRunning" class="w-4 h-4" />
        <heroicons-outline:play v-else class="w-4 h-4" />
      </template>
      <template #default>
        <template v-if="isRunning">
          {{ $t("task.checking") }}
        </template>
        <template v-else>
          {{ $t("task.run-checks") }}
        </template>
      </template>
    </NButton>
  </div>
</template>

<script lang="ts" setup>
import { ButtonProps, NButton } from "naive-ui";
import { ref } from "vue";
import { CSSProperties } from "vue";
import { sqlServiceClient } from "@/grpcweb";
import { ComposedDatabase } from "@/types";
import { Advice } from "@/types/proto/v1/sql_service";

const props = defineProps<{
  statement: string;
  database: ComposedDatabase;
  buttonProps?: ButtonProps;
  buttonStyle?: string | CSSProperties;
}>();

const isRunning = ref(false);
const advices = ref<Advice[]>();

const runChecks = async () => {
  const { statement, database } = props;
  isRunning.value = true;
  if (!advices.value) {
    advices.value = [];
  }
  try {
    const result = await sqlServiceClient.check({
      statement,
      database: database.name,
    });
    advices.value = result.advices;
  } finally {
    isRunning.value = false;
  }
};
</script>
