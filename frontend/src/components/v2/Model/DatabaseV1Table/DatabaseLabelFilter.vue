<template>
  <div class="flex flex-row flex-wrap gap-x-2 gap-y-2 text-sm select-none">
    <div
      v-for="kv in distinctLabelList"
      :key="`${kv.key}:${kv.value}`"
      class="px-2 py-0.5 rounded cursor-pointer border text-control"
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
</template>

<script setup lang="ts">
import { orderBy, uniqBy } from "lodash-es";
import { computed } from "vue";
import { ComposedDatabase } from "@/types";

type KV = { key: string; value: string };

const props = defineProps<{
  selected: KV[];
  databaseList: ComposedDatabase[];
}>();

const emit = defineEmits<{
  (event: "update:selected", selected: KV[]): void;
}>();

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
</script>
