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
import { computed, h } from "vue";
import { NSelect, SelectOption } from "naive-ui";
import { useI18n } from "vue-i18n";

import { Stage } from "@/types/proto/v1/rollout_service";
import { useIssueContext } from "../logic";
import { first } from "lodash-es";

type StageSelectOption = SelectOption & {
  stage: Stage;
};

const { t } = useI18n();
const { isCreating, issue, activeStage, selectedStage, events } =
  useIssueContext();

const stageList = computed(() => issue.value.rolloutEntity.stages);

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
    !isCreating.value && stage === activeStage.value
      ? t("issue.stage-select.active", { name: stage.title })
      : stage.title;
  return h("div", { class: "flex items-center gap-x-1" }, [
    h("span", {}, text),
  ]);
};

const handleSelectStage = (name: string) => {
  const stage = stageList.value.find((s) => s.name === name);
  if (stage) {
    const task = first(stage.tasks);
    if (task) {
      events.emit("select-task", { task });
    }
  }
};
</script>
