<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <div class="deployment-stage flex w-full relative">
    <div
      v-if="allowEdit"
      class="reorder flex flex-col items-center justify-start pr-2 pb-4 pt-5"
    >
      <NButton v-if="index > 0" text @click="$emit('prev')">
        <heroicons-solid:arrow-circle-up class="w-6 h-6" />
      </NButton>
      <NButton v-if="index < max - 1" text @click="$emit('next')">
        <heroicons-solid:arrow-circle-down class="w-6 h-6" />
      </NButton>
    </div>
    <div
      class="main flex-1 flex flex-col gap-y-2 w-full overflow-x-hidden"
      :class="[layout === 'compact' ? 'my-2' : 'my-4']"
    >
      <h3 v-if="showHeader">
        <template v-if="allowEdit">
          <NInput
            v-model:value="deployment.title"
            :placeholder="$t('deployment-config.name-placeholder')"
            style="width: 14rem"
          />
        </template>
        <template v-else>
          {{ deployment.title }}
        </template>
      </h3>
      <div class="flex flex-col gap-y-2 overflow-hidden">
        <div
          v-for="(selector, j) in selectors"
          :key="j"
          class="flex content-start overflow-hidden"
        >
          <SelectorItemV2
            :editable="allowEdit"
            :selector="selector"
            :selectors="selectors"
            :index="j"
            :database-list="databaseList"
            @remove="removeSelector(selector)"
          />
        </div>
      </div>
      <NButton v-if="allowEdit" class="self-start" @click="addSelector">
        {{ $t("deployment-config.add-selector") }}
      </NButton>
    </div>

    <NButton
      v-if="allowEdit"
      class="absolute right-0 top-0"
      quaternary
      style="--n-padding: 10px"
      @click="$emit('remove')"
    >
      <XIcon class="w-4 h-4" />
    </NButton>
  </div>
</template>

<script lang="ts" setup>
import { head, without } from "lodash-es";
import { XIcon } from "lucide-vue-next";
import { NButton, NInput } from "naive-ui";
import { computed, PropType } from "vue";
import { ComposedDatabase } from "@/types";
import {
  LabelSelectorRequirement,
  OperatorType,
  ScheduleDeployment,
} from "@/types/proto/v1/project_service";
import { getAvailableDeploymentConfigMatchSelectorKeyList } from "@/utils";
import SelectorItemV2 from "./SelectorItemV2.vue";

const props = defineProps({
  deployment: {
    type: Object as PropType<ScheduleDeployment>,
    required: true,
  },
  index: {
    type: Number,
    default: -1,
  },
  max: {
    type: Number,
    default: -1,
  },
  allowEdit: {
    type: Boolean,
    default: false,
  },
  databaseList: {
    type: Array as PropType<ComposedDatabase[]>,
    default: () => [],
  },
  showHeader: {
    type: Boolean,
    default: true,
  },
  layout: {
    type: String as PropType<"default" | "compact">,
    default: "default",
  },
});

defineEmits<{
  (event: "remove"): void;
  (event: "prev"): void;
  (event: "next"): void;
}>();

const selectors = computed(() => {
  return props.deployment.spec?.labelSelector?.matchExpressions ?? [];
});

const keys = computed(() => {
  return getAvailableDeploymentConfigMatchSelectorKeyList(
    props.databaseList,
    true /* withVirtualLabelKeys */,
    true /* sort */
  );
});

const removeSelector = (selector: LabelSelectorRequirement) => {
  const index = selectors.value.indexOf(selector);
  if (index >= 0) {
    selectors.value.splice(index, 1);
  }
};

const addSelector = () => {
  const usedKeys = selectors.value.map((s) => s.key);
  const unusedKeys = without(keys.value, ...usedKeys);
  const labelKeys = without(keys.value, "environment");
  const suggestedNextKey =
    head(unusedKeys) ?? head(labelKeys) ?? head(keys.value) ?? "";

  const rule: LabelSelectorRequirement = {
    key: suggestedNextKey,
    operator: OperatorType.OPERATOR_TYPE_IN,
    values: [],
  };
  selectors.value.push(rule);
};
</script>
