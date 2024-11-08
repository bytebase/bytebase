<template>
  <div class="w-full flex grow overflow-auto bg-zinc-100 p-6 rounded-md">
    <div class="relative w-auto flex flex-row items-start justify-start gap-6">
      <div class="absolute top-5 border-2 w-full"></div>
      <div
        v-for="stage in stages"
        :key="stage.name"
        class="!w-80 bg-white z-[1] rounded-lg p-1 hover:shadow"
        content-class="flex flex-col gap-2"
      >
        <p class="textlabel px-2 mt-2 mb-1">{{ stage.title }}</p>
        <div
          v-for="task in stage.tasks"
          :key="task.name"
          class="w-full flex flex-row items-center justify-start truncate gap-2 px-2 py-1 rounded-md cursor-pointer hover:bg-zinc-50"
          @click="onTaskClick(task)"
        >
          <TaskStatus :status="task.status" />
          <div class="truncate">
            <div class="w-auto flex flex-row items-center text-sm truncate">
              <InstanceV1EngineIcon
                class="inline-block mr-1"
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
              <NTag round size="tiny">{{ semanticTaskType(task.type) }}</NTag>
              <NTag
                v-if="extractSchemaVersionFromTask(task)"
                round
                size="tiny"
                >{{ extractSchemaVersionFromTask(task) }}</NTag
              >
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ChevronRightIcon } from "lucide-vue-next";
import { NTag } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { semanticTaskType } from "@/components/IssueV1";
import { InstanceV1EngineIcon } from "@/components/v2";
import { type Task } from "@/types/proto/v1/rollout_service";
import { extractSchemaVersionFromTask } from "@/utils";
import { useRolloutDetailContext } from "../context";
import { databaseForTask } from "../utils";
import TaskStatus from "./kits/TaskStatus.vue";

const router = useRouter();
const { rollout } = useRolloutDetailContext();

const stages = computed(() => rollout.value.stages);

const onTaskClick = (task: Task) => {
  router.push(`/${task.name}`);
};
</script>
