<template>
  <template v-if="showSessionTables">
    <PostgresSessionTable
      v-if="postgresSession"
      :task-run-session="postgresSession"
    />

    <div v-if="isLoading" class="flex justify-center items-center py-10">
      <BBSpin />
    </div>
  </template>
  <div v-else class="text-control-placeholder">
    {{ $t("issue.task-run.task-run-session.no-session-found") }}
  </div>
</template>

<script setup lang="tsx">
import { computedAsync } from "@vueuse/core";
import { computed, ref } from "vue";
import { BBSpin } from "@/bbkit";
import { rolloutServiceClient } from "@/grpcweb";
import { TaskRun_Status, type TaskRun } from "@/types/proto/api/v1alpha/rollout_service";
import PostgresSessionTable from "./PostgresSessionTable.vue";

const props = defineProps<{
  taskRun: TaskRun;
}>();

const isLoading = ref(false);

const showSessionTables = computed(
  () => props.taskRun.status === TaskRun_Status.RUNNING
);

const taskRunSession = computedAsync(
  () => {
    if (!showSessionTables.value) {
      return undefined;
    }
    return rolloutServiceClient.getTaskRunSession(
      {
        parent: props.taskRun.name,
      },
      {
        silent: true,
      }
    );
  },
  undefined,
  {
    evaluating: isLoading,
  }
);

const postgresSession = computed(() => taskRunSession.value?.postgres);
</script>
