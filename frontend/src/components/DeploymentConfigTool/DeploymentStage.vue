<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <div class="deployment-stage flex w-full relative">
    <div
      v-if="allowEdit"
      class="reorder flex flex-col items-center justify-start pr-2 py-4"
    >
      <button
        v-if="index > 0"
        class="text-control hover:text-control-hover"
        @click="$emit('prev')"
      >
        <heroicons-solid:arrow-circle-up class="w-6 h-6" />
      </button>
      <button
        v-if="index < max - 1"
        class="text-control hover:text-control-hover"
        @click="$emit('next')"
      >
        <heroicons-solid:arrow-circle-down class="w-6 h-6" />
      </button>
    </div>
    <div
      class="main flex-1 space-y-2 w-full"
      :class="[layout === 'compact' ? 'py-2' : 'py-4']"
    >
      <h3 v-if="showHeader">
        <template v-if="allowEdit">
          <input
            v-model="deployment.title"
            type="text"
            :placeholder="$t('deployment-config.name-placeholder')"
            class="rounded-md border-control-border focus:ring-control focus:border-control disabled:bg-gray-50 h-8 py-0 text-sm"
          />
        </template>
        <template v-else>
          {{ deployment.title }}
        </template>
      </h3>
      <div class="space-y-2 overflow-hidden">
        <div
          v-for="(selector, j) in deployment.spec?.labelSelector
            ?.matchExpressions"
          :key="j"
          class="flex content-start"
        >
          <SelectorItem
            :editable="allowEdit"
            :selector="selector"
            :database-list="databaseList"
            @remove="removeSelector(selector)"
          />
        </div>
      </div>
      <button v-if="allowEdit" class="btn-normal btn-add" @click="addSelector">
        {{ $t("deployment-config.add-selector") }}
      </button>
    </div>

    <span
      v-if="allowEdit"
      class="absolute right-2 top-2 p-1 text-control cursor-pointer hover:bg-gray-200 rounded-sm"
      @click="$emit('remove')"
    >
      <heroicons:solid:x class="w-4 h-4" />
    </span>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import {
  LabelSelectorRequirement,
  OperatorType,
  ScheduleDeployment,
} from "@/types/proto/v1/project_service";
import { PRESET_LABEL_KEYS, RESERVED_LABEL_KEYS } from "@/utils";
import { ComposedDatabase } from "../../types";
import SelectorItem from "./SelectorItem.vue";

export default defineComponent({
  name: "DeploymentStage",
  components: { SelectorItem },
  props: {
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
  },
  emits: ["remove", "prev", "next"],
  setup(props) {
    const keys = computed(() => {
      return [...RESERVED_LABEL_KEYS, ...PRESET_LABEL_KEYS];
    });

    const removeSelector = (selector: LabelSelectorRequirement) => {
      const array =
        props.deployment.spec?.labelSelector?.matchExpressions ?? [];
      const index = array.indexOf(selector);
      if (index >= 0) {
        array.splice(index, 1);
      }
    };

    const addSelector = () => {
      const array =
        props.deployment.spec?.labelSelector?.matchExpressions ?? [];
      const rule: LabelSelectorRequirement = {
        key: keys.value[0] ?? "",
        operator: OperatorType.OPERATOR_TYPE_IN,
        values: [],
      };
      array.push(rule);
    };

    return {
      removeSelector,
      addSelector,
    };
  },
});
</script>

<style scoped lang="postcss">
.btn-add {
  @apply py-1.5 !important;
}
</style>
