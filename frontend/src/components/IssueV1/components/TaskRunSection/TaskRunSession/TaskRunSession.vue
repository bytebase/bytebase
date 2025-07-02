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
import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { silentContextKey } from "@/grpcweb/context-key";
import { GetTaskRunSessionRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { BBSpin } from "@/bbkit";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { TaskRun_Status, type TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import PostgresSessionTable from "./PostgresSessionTable.vue";

const props = defineProps<{
  taskRun: TaskRun;
}>();

const isLoading = ref(false);

const showSessionTables = computed(
  () => props.taskRun.status === TaskRun_Status.RUNNING
);

const taskRunSession = computedAsync(
  async () => {
    if (!showSessionTables.value) {
      return undefined;
    }
    const request = create(GetTaskRunSessionRequestSchema, {
      parent: props.taskRun.name,
    });
    const response = await rolloutServiceClientConnect.getTaskRunSession(
      request,
      {
        contextValues: createContextValues().set(silentContextKey, true),
      }
    );
    return response;
  },
  undefined,
  {
    evaluating: isLoading,
  }
);

const postgresSession = computed(() => {
  const session = taskRunSession.value;
  if (session?.session?.case === "postgres") {
    return session.session.value;
  }
  return undefined;
});
</script>
