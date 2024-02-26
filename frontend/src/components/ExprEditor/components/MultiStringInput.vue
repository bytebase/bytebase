<template>
  <NSelect
    :value="value"
    :filterable="true"
    :multiple="true"
    :tag="true"
    :show-arrow="false"
    :show="false"
    :consistent-menu-width="false"
    :placeholder="$t('cel.condition.input-value-press-enter')"
    :disabled="!allowAdmin"
    max-tag-count="responsive"
    size="small"
    style="min-width: 16rem; width: auto; overflow-x: hidden"
    @update:value="onUpdate"
  />
</template>

<script lang="ts" setup>
import { uniq } from "lodash-es";
import { NSelect } from "naive-ui";
import { useExprEditorContext } from "../context";

defineProps<{
  value: string[];
}>();

const emit = defineEmits<{
  (event: "update:value", value: string[]): void;
}>();

const context = useExprEditorContext();
const { allowAdmin } = context;

const onUpdate = (values: string[]) => {
  emit(
    "update:value",
    uniq(
      values
        .join(",")
        .split(",")
        .map((val) => val.trim())
        .filter((val) => val)
    )
  );
};
</script>
