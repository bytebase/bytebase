<template>
  <div class="flex flex-row items-center gap-2">
    <slot name="result" :advices="advices" :is-running="isRunning" />

    <NTooltip :disabled="!errors || errors.length === 0">
      <template #trigger>
        <NButton
          style="--n-padding: 0 14px 0 10px"
          :disabled="disabled"
          :style="buttonStyle"
          tag="div"
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
      </template>
      <template #default>
        <ErrorList :errors="errors ?? []" />
      </template>
    </NTooltip>
  </div>
</template>

<script lang="ts" setup>
import { ButtonProps, NButton, NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { CSSProperties } from "vue";
import { sqlServiceClient } from "@/grpcweb";
import { ComposedDatabase } from "@/types";
import { Advice } from "@/types/proto/v1/sql_service";
import ErrorList from "../misc/ErrorList.vue";

const props = defineProps<{
  statement: string;
  database: ComposedDatabase;
  buttonProps?: ButtonProps;
  buttonStyle?: string | CSSProperties;
  errors?: string[];
}>();

const isRunning = ref(false);
const advices = ref<Advice[]>();

const disabled = computed(() => {
  if (!props.statement) return true;
  return props.errors && props.errors.length > 0;
});

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
