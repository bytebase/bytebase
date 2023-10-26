<template>
  <div v-bind="$attrs">
    <div
      ref="containerRef"
      class="flex flex-row flex-wrap gap-x-2 gap-y-2 text-sm select-none relative"
      :style="style"
    >
      <div
        v-for="kv in distinctLabelList"
        :key="`${kv.key}:${kv.value}`"
        class="label-item px-2 py-0.5 rounded cursor-pointer border text-control"
        :class="
          isKVSelected(kv)
            ? 'border-accent-tw bg-accent-tw/10'
            : 'bg-gray-100 hover:bg-accent-tw/10 border-gray-100 hover:border-gray-200'
        "
        @click="toggleSelection(kv, !isKVSelected(kv))"
      >
        <span>{{ kv.key }}</span>
        <span>:</span>
        <span :class="!kv.value && 'text-control-placeholder'">
          {{ kv.value || $t("label.empty-label-value") }}
        </span>
      </div>
    </div>

    <div
      v-if="shouldCollapse && !expanded"
      class="flex flex-row justify-end mt-2 text-sm"
    >
      <div class="normal-link" @click="expanded = true">
        {{ $t("common.show-all") }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useElementSize } from "@vueuse/core";
import { orderBy, uniq, uniqBy } from "lodash-es";
import { computed, ref, watch } from "vue";
import { CSSProperties } from "vue";
import { ComposedDatabase } from "@/types";

type KV = { key: string; value: string };
const ROW_HEIGHT = 26;
const ROW_GAP = 8;

const props = withDefaults(
  defineProps<{
    selected: KV[];
    databaseList: ComposedDatabase[];
    maxRows?: number;
  }>(),
  {
    maxRows: 3,
  }
);

const emit = defineEmits<{
  (event: "update:selected", selected: KV[]): void;
}>();

const containerRef = ref<HTMLDivElement>();
const containerSize = useElementSize(containerRef);
const rowYCoords = ref<number[]>([]);
const expanded = ref(false);

const distinctLabelList = computed(() => {
  const list = props.databaseList.flatMap((db) => {
    return Object.keys(db.labels).map<KV>((key) => ({
      key,
      value: db.labels[key],
    }));
  });
  const distinctList = uniqBy(list, (kv) => `${kv.key}:${kv.value}`);
  const sortedList = orderBy(
    distinctList,
    [
      (kv) => kv.key, // by key ASC
      (kv) => (kv.value ? -1 : 1), // then put empty values at last
      (kv) => kv.value, // then by value ASC
    ],
    ["asc", "asc", "asc"]
  );
  return sortedList;
});
const shouldCollapse = computed(() => {
  const rowCount = rowYCoords.value.length;
  return rowCount > props.maxRows;
});
const style = computed(() => {
  const style: CSSProperties = {};
  if (shouldCollapse.value) {
    if (expanded.value) {
      //
    } else {
      style.overflowY = "hidden";
      const maxHeight =
        props.maxRows * ROW_HEIGHT + (props.maxRows - 1) * ROW_GAP;
      style.maxHeight = `${maxHeight}px`;
    }
  }

  return style;
});

const isKVSelected = (kv: KV) => {
  return !!props.selected.find(
    ({ key, value }) => key === kv.key && value === kv.value
  );
};

const toggleSelection = (kv: KV, checked: boolean) => {
  const index = props.selected.findIndex(
    ({ key, value }) => key === kv.key && value === kv.value
  );
  if (checked && index < 0) {
    emit("update:selected", [...props.selected, kv]);
  } else if (!checked && index >= 0) {
    const updated = [...props.selected];
    updated.splice(index, 1);
    emit("update:selected", updated);
  }
};

watch(
  [containerSize.width, containerSize.height, distinctLabelList, props.maxRows],
  ([containerWidth, containerHeight, list]) => {
    const containerElement = containerRef.value;
    if (!containerElement) {
      rowYCoords.value = [];
      return;
    }

    const children = Array.from(
      containerElement.querySelectorAll(".label-item")
    );
    const distinctChildrenTops = uniq(
      children.map((element) => element.getBoundingClientRect().top)
    );
    rowYCoords.value = distinctChildrenTops;
  },
  { immediate: true }
);

watch(
  shouldCollapse,
  (shouldCollapse) => {
    if (!shouldCollapse) {
      expanded.value = false;
    }
  },
  { immediate: true }
);
</script>
