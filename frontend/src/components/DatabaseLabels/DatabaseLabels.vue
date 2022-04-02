<template>
  <div class="flex gap-2">
    <DatabaseLabel
      v-for="(label, i) in labelList"
      :key="i"
      :label="label"
      :editable="isEditableLabel(label)"
      :available-label-list="availableLabelList"
      @remove="removeLabel(i)"
    />
    <template v-if="editable">
      <NPopover trigger="hover" :disabled="allowAdd">
        <template #trigger>
          <button
            class="add-button"
            :class="{ disabled: !allowAdd }"
            @click="addLabel"
          >
            <heroicons-solid:plus class="w-4 h-4" />
          </button>
        </template>

        <div class="text-red-600 whitespace-nowrap">
          {{
            $t("label.error.max-label-count-exceeded", {
              count: MAX_DATABASE_LABEL_COUNT,
            })
          }}
        </div>
      </NPopover>
    </template>
  </div>
</template>

<script lang="ts">
/* eslint-disable vue/no-mutating-props */

import { computed, defineComponent, PropType, watchEffect } from "vue";
import { DatabaseLabel } from "../../types";
import { NPopover } from "naive-ui";
import { isReservedLabel, isReservedDatabaseLabel } from "../../utils";
import { useLabelStore } from "@/store";
import { storeToRefs } from "pinia";

const MAX_DATABASE_LABEL_COUNT = 4;

export default defineComponent({
  name: "DatabaseLabels",
  components: { NPopover },
  props: {
    labelList: {
      type: Array as PropType<DatabaseLabel[]>,
      default: () => [],
    },
    editable: {
      type: Boolean,
      default: false,
    },
  },
  setup(props) {
    const labelStore = useLabelStore();

    const allowAdd = computed(
      () => props.labelList.length < MAX_DATABASE_LABEL_COUNT
    );

    const prepareLabelList = () => {
      // need not to fetchLabelList if not editable
      if (!props.editable) return;
      labelStore.fetchLabelList();
    };
    watchEffect(prepareLabelList);

    const { labelList } = storeToRefs(labelStore);

    const availableLabelList = computed(() =>
      labelList.value.filter((label) => !isReservedLabel(label))
    );

    const addLabel = () => {
      if (!allowAdd.value) return;

      const key = labelList.value[0]?.key || "";
      const value = labelList.value[0]?.valueList[0] || "";
      props.labelList.push({
        key,
        value,
      });
    };

    const removeLabel = (index: number) => {
      props.labelList.splice(index, 1);
    };

    const isEditableLabel = (label: DatabaseLabel): boolean => {
      if (labelList.value.length === 0) {
        // not ready yet, disable editing temporarily
        // this also avoid some UI blinking
        return false;
      }

      return props.editable && !isReservedDatabaseLabel(label, labelList.value);
    };

    return {
      MAX_DATABASE_LABEL_COUNT,
      availableLabelList,
      isEditableLabel,
      allowAdd,
      addLabel,
      removeLabel,
    };
  },
});
</script>

<style scoped lang="postcss">
.add-button {
  @apply h-6 px-1 py-1 inline-flex items-center
    rounded bg-white border border-control-border
    hover:bg-control-bg-hover
    cursor-pointer;
}
.add-button.disabled {
  @apply cursor-not-allowed bg-control-bg;
}
</style>
