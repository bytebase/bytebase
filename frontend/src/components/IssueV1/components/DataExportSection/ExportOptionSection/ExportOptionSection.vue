<template>
  <div class="w-full flex flex-col">
    <div class="w-full flex flex-row items-center mb-1">
      <span class="textlabel mr-4">{{ $t("issue.data-export.options") }}</span>
    </div>
    <div class="w-full h-8 flex flex-row justify-start items-center">
      <span class="textinfolabel inline-block mr-2 !min-w-[64px]">{{
        $t("issue.data-export.format")
      }}</span>
      <ExportFormatSelector v-model:format="state.config.format" />
    </div>
    <div class="w-full h-8 flex flex-row justify-start items-center">
      <span class="textinfolabel inline-block mr-2 !min-w-[64px]">{{
        $t("issue.data-export.encrypt")
      }}</span>
      <ExportPasswordInputer v-model:password="state.config.password" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { computed, watch, reactive } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import {
  Plan_Spec,
  Plan_ExportDataConfig,
} from "@/types/proto/v1/rollout_service";
import ExportFormatSelector from "./ExportFormatSelector.vue";
import ExportPasswordInputer from "./ExportPasswordInputer.vue";

interface LocalState {
  config: Plan_ExportDataConfig;
}

const { issue } = useIssueContext();

const spec = computed(
  () =>
    head(issue.value.planEntity?.steps.flatMap((step) => step.specs)) ||
    Plan_Spec.fromPartial({})
);

const state = reactive<LocalState>({
  config: spec.value.exportDataConfig || Plan_ExportDataConfig.fromPartial({}),
});

watch(
  () => state.config,
  () => {
    spec.value.exportDataConfig = Plan_ExportDataConfig.fromPartial({
      ...spec.value.exportDataConfig,
      format: state.config.format,
      password: state.config.password,
    });
  },
  { deep: true }
);
</script>
