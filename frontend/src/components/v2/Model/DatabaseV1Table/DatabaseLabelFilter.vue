<template>
  <NCascader
    class="bb-database-label-filter"
    :value="cascaderValues"
    :options="options"
    :placeholder="$t('label.filter-by-label')"
    :render-label="renderLabel"
    :multiple="true"
    :show-path="true"
    :check-strategy="'child'"
    :max-tag-count="'responsive'"
    :expand-trigger="'click'"
    :virtual-scroll="true"
    :filterable="true"
    :menu-props="{
      style: '--n-column-width: max-content',
    }"
    :separator="':'"
    style="width: 12rem"
    @update:value="updateCascaderValues"
  />
</template>

<script setup lang="ts">
import { orderBy, uniqBy } from "lodash-es";
import { CascaderOption, NCascader } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { ComposedDatabase } from "@/types";
import { groupBy } from "@/utils";

type KV = { key: string; value: string };
type KeyOption = CascaderOption & {
  type: "key";
  key: string;
  value: string;
};
type KeyValueOption = CascaderOption & {
  type: "kv";
  kv: KV;
  value: string;
};

const props = defineProps<{
  selected: KV[];
  databaseList: ComposedDatabase[];
}>();

const emit = defineEmits<{
  (event: "update:selected", selected: KV[]): void;
}>();

const { t } = useI18n();

const cascaderValueOfKV = (kv: KV) => {
  return JSON.stringify(kv);
};
const distinctKVList = computed(() => {
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
const options = computed(() => {
  const groups = groupBy(distinctKVList.value, (kv) => kv.key);
  return Array.from(groups.entries()).map<KeyOption>(([key, group]) => {
    const children = group.map<KeyValueOption>((kv) => ({
      type: "kv",
      kv,
      value: cascaderValueOfKV(kv),
      label: kv.value || t("label.empty-label-value"),
    }));
    return {
      type: "key",
      key,
      value: key,
      label: key,
      children,
    };
  });
});
const renderLabel = (option: KeyOption | KeyValueOption) => {
  if (option.type === "key") {
    const { key } = option;
    return key;
  }
  if (option.type === "kv") {
    const { kv } = option;
    if (!kv.value) {
      return h(
        "span",
        {
          class: "text-control-placeholder",
        },
        t("label.empty-label-value")
      );
    }
    return kv.value;
  }
  console.error("should never reach this line", option);
  return "";
};

const cascaderValues = computed(() => {
  return props.selected.map(cascaderValueOfKV);
});

const updateCascaderValues = (
  values: string[],
  options: (KeyOption | KeyValueOption)[]
) => {
  const kvOptions = options.filter(
    (opt) => opt.type === "kv"
  ) as KeyValueOption[];
  emit(
    "update:selected",
    kvOptions.map((opt) => opt.kv)
  );
};
</script>
