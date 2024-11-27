<template>
  <div class="flex items-center gap-x-2">
    <div class="textlabel">
      {{ $t("common.stage") }}
    </div>
    <NSelect
      :value="selectedStage.name"
      :options="options"
      :render-label="renderLabel"
      style="width: 12rem"
      @update-value="handleSelectStage"
    />
  </div>
</template>

<script setup lang="ts">
import { first } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { EMPTY_TASK_NAME } from "@/types";
import type { Stage } from "@/types/proto/v1/rollout_service";
import { activeTaskInStageV1 } from "@/utils";
import { useIssueContext } from "../../../logic";

type StageSelectOption = SelectOption & {
  stage: Stage;
};

const { t } = useI18n();
const { isCreating, issue, selectedStage, events } =
  useIssueContext();

const stageList = computed(() => issue.value.rolloutEntity?.stages || []);

const options = computed(() => {
  return stageList.value.map<StageSelectOption>((stage) => ({
    value: stage.name,
    label: stage.title,
    stage,
  }));
});

const renderLabel = (option: SelectOption) => {
  const { stage } = option as StageSelectOption;

  const text =
    !isCreating.value && stage === selectedStage.value
      ? t("issue.stage-select.current", { name: stage.title })
      : stage.title;
  return h("div", { class: "flex items-center gap-x-1" }, [
    h("span", {}, text),
  ]);
};

const activeOrFirstTaskInStage = (stage: Stage) => {
  if (isCreating.value) {
    return first(stage.tasks);
  }
  const activeTask = activeTaskInStageV1(stage);
  if (activeTask.name === EMPTY_TASK_NAME) {
    return first(stage.tasks);
  }
  return activeTask;
};

const handleSelectStage = (name: string) => {
  const stage = stageList.value.find((s) => s.name === name);
  if (stage === selectedStage.value) return;

  if (stage) {
    const task = activeOrFirstTaskInStage(stage);
    if (task) {
      events.emit("select-task", { task });
    }
  }
};
</script>
