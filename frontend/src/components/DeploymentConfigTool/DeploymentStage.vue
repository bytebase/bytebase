<template>
  <!-- eslint-disable vue/no-mutating-props -->

  <div class="deployment-stage flex w-full relative">
    <div
      v-if="allowEdit"
      class="reorder flex flex-col items-center justify-between pr-2 py-0"
    >
      <button
        class="text-control hover:text-control-hover"
        :class="[index > 0 ? 'visible' : 'invisible']"
        @click="$emit('prev')"
      >
        <heroicons-solid:arrow-circle-up class="w-6 h-6" />
      </button>
      <button
        class="text-control hover:text-control-hover"
        :class="[index < max - 1 ? 'visible' : 'invisible']"
        @click="$emit('next')"
      >
        <heroicons-solid:arrow-circle-down class="w-6 h-6" />
      </button>
    </div>
    <div class="main flex-1 space-y-2 py-2 w-full">
      <h3 v-if="showHeader" class="text-lg leading-6 font-medium text-main">
        <template v-if="allowEdit">
          <input
            v-model="deployment.name"
            type="text"
            :placeholder="$t('deployment-config.name-placeholder')"
            class="text-main rounded-md border-control-border focus:ring-control focus:border-control disabled:bg-gray-50"
          />
        </template>
        <template v-else>
          {{ deployment.name }}
        </template>
      </h3>
      <div class="space-y-2 overflow-hidden">
        <div
          v-for="(selector, j) in deployment.spec.selector.matchExpressions"
          :key="j"
          class="flex content-start"
        >
          <SelectorItem
            :editable="allowEdit"
            :selector="selector"
            :label-list="labelList"
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
import { defineComponent, PropType } from "vue";
import {
  AvailableLabel,
  Database,
  Deployment,
  LabelSelectorRequirement,
} from "../../types";
import SelectorItem from "./SelectorItem.vue";

export default defineComponent({
  name: "DeploymentStage",
  components: { SelectorItem },
  props: {
    deployment: {
      type: Object as PropType<Deployment>,
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
      type: Array as PropType<Database[]>,
      default: () => [],
    },
    labelList: {
      type: Array as PropType<AvailableLabel[]>,
      default: () => [],
    },
    showHeader: {
      type: Boolean,
      default: true,
    },
  },
  emits: ["remove", "prev", "next"],
  setup(props) {
    const removeSelector = (selector: LabelSelectorRequirement) => {
      const array = props.deployment.spec.selector.matchExpressions;
      const index = array.indexOf(selector);
      if (index >= 0) {
        array.splice(index, 1);
      }
    };

    const addSelector = () => {
      const array = props.deployment.spec.selector.matchExpressions;
      const label = props.labelList[0];
      const rule: LabelSelectorRequirement = {
        key: label?.key || "",
        operator: "In",
        values: [],
      };
      if (label && label.valueList.length > 0) {
        rule.values.push(label.valueList[0]);
      }
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
