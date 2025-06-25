<template>
  <div
    class="!w-80 bg-white z-[1] rounded-lg p-1 hover:shadow"
    :class="
      twMerge(isCreated ? 'bg-white' : 'bg-zinc-50 border-2 border-dashed')
    "
  >
    <div
      class="w-full flex flex-row justify-between items-center gap-2 px-2 pt-2 pb-1"
    >
      <p class="textlabel">
        {{ environmentStore.getEnvironmentByName(stage.environment).title }}
        <NTag v-if="!isCreated" round size="tiny">Not created</NTag>
      </p>
      <div>
        <NPopconfirm
          v-if="!isCreated"
          :negative-text="null"
          :positive-text="$t('common.confirm')"
          :positive-button-props="{ size: 'tiny' }"
          @positive-click="createRolloutToStage"
        >
          <template #trigger>
            <NButton text size="small">
              <template #icon>
                <CirclePlayIcon />
              </template>
            </NButton>
          </template>
          {{ $t("common.confirm-and-add") }}
        </NPopconfirm>
      </div>
    </div>
    <NVirtualList
      style="max-height: 100vh"
      :items="stage.tasks"
      :item-size="56"
    >
      <template #default="{ item: task }: { item: Task }">
        <div
          :key="task.name"
          class="w-full border-t border-zinc-50 flex flex-col items-start justify-start truncate px-2 py-1 h-14"
          :class="
            isCreated && 'cursor-pointer hover:bg-zinc-50 hover:shadow-sm'
          "
          @click="onTaskClick(task)"
        >
          <div class="w-full flex flex-row items-center text-sm truncate">
            <TaskStatus :status="task.status" size="small" />
            <InstanceV1EngineIcon
              class="inline-block ml-2 mr-1"
              :instance="
                databaseForTask(rollout.projectEntity, task).instanceResource
              "
            />
            <span class="truncate">
              {{
                databaseForTask(rollout.projectEntity, task).instanceResource
                  .title
              }}
            </span>
            <ChevronRightIcon class="inline opacity-60 w-4 shrink-0" />
            <span class="truncate">
              {{ databaseForTask(rollout.projectEntity, task).databaseName }}
            </span>
          </div>
          <p class="space-x-1 mt-0.5 leading-4">
            <NTooltip>
              <template #trigger>
                <NTag round size="tiny">{{ semanticTaskType(task.type) }}</NTag>
              </template>
              {{ $t("common.type") }}
            </NTooltip>
            <NTooltip v-if="extractSchemaVersionFromTask(task)">
              <template #trigger>
                <NTag round size="tiny">
                  {{ extractSchemaVersionFromTask(task) }}
                </NTag>
              </template>
              {{ $t("common.version") }}
            </NTooltip>
          </p>
        </div>
      </template>
    </NVirtualList>
  </div>
</template>

<script lang="ts" setup>
import { ChevronRightIcon, CirclePlayIcon } from "lucide-vue-next";
import { NTag, NVirtualList, NTooltip, NButton, NPopconfirm } from "naive-ui";
import { twMerge } from "tailwind-merge";
import { computed } from "vue";
import { create } from "@bufbuild/protobuf";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { useRouter } from "vue-router";
import { semanticTaskType } from "@/components/IssueV1";
import { InstanceV1EngineIcon } from "@/components/v2";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { useEnvironmentV1Store } from "@/store";
import { Stage, type Task } from "@/types/proto/v1/rollout_service";
import { extractSchemaVersionFromTask } from "@/utils";
import { useRolloutDetailContext } from "../../context";
import { databaseForTask } from "../../utils";
import TaskStatus from "./../kits/TaskStatus.vue";

const props = defineProps<{
  stage: Stage;
}>();

const router = useRouter();
const { project, rollout, emmiter } = useRolloutDetailContext();
const environmentStore = useEnvironmentV1Store();

const isCreated = computed(() => {
  return rollout.value.stages.some(
    (stage) => stage.environment === props.stage.environment
  );
});

const onTaskClick = (task: Task) => {
  if (!isCreated.value) {
    return;
  }

  router.push(`/${task.name}`);
};

const createRolloutToStage = async () => {
  const request = create(CreateRolloutRequestSchema, {
    parent: project.value.name,
    rollout: {
      plan: rollout.value.plan,
    },
    target: props.stage.environment,
  });
  await rolloutServiceClientConnect.createRollout(request);
  emmiter.emit("task-status-action");
};
</script>
