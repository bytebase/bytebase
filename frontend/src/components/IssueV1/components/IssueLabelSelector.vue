<template>
  <NSelect
    :multiple="true"
    :options="options"
    :disabled="disabled"
    :size="size"
    :consistent-menu-width="true"
    :max-tag-count="maxTagCount"
    :render-label="renderLabel"
    :render-tag="renderTag"
    :value="issueLabels"
    @update:value="onLablesUpdate"
  />
</template>

<script setup lang="ts">
import { NCheckbox, NSelect, NTag } from "naive-ui";
import type { SelectOption } from "naive-ui";
import type { SelectBaseOption } from "naive-ui/lib/select/src/interface";
import { computed, h } from "vue";
import { Label } from "@/types/proto/v1/project_service";

type IsseuLabelOption = SelectOption & {
  value: string;
  color: string;
};

const props = withDefaults(
  defineProps<{
    disabled: boolean;
    selected: string[];
    labels: Label[];
    size: "small" | "medium" | "large";
    maxTagCount: number | "responsive";
  }>(),
  {
    size: "medium",
    maxTagCount: "responsive",
  }
);

const emit = defineEmits<{
  (event: "update:selected", selected: string[]): void;
}>();

const issueLabels = computed(() => {
  const pool = new Set(props.labels.map((label) => label.value));
  return props.selected.filter((label) => pool.has(label));
});

const options = computed(() => {
  return props.labels.map<IsseuLabelOption>((label) => ({
    label: label.value,
    value: label.value,
    color: label.color,
  }));
});

const onLablesUpdate = async (labels: string[]) => {
  emit("update:selected", labels);
};

const renderLabel = (option: IsseuLabelOption, selected: boolean) => {
  const { color, value } = option as IsseuLabelOption;
  return h("div", { class: "flex items-center gap-x-2" }, [
    h(NCheckbox, { checked: selected, size: "small" }),
    h("div", {
      class: "w-4 h-4 rounded cursor-pointer relative",
      style: `background-color: ${color};`,
      onClick: () => {},
    }),
    value,
  ]);
};

const renderTag = ({
  option,
  handleClose,
}: {
  option: SelectBaseOption;
  handleClose: () => void;
}) => {
  const { color, value } = option as IsseuLabelOption;

  return h(
    NTag,
    {
      size: props.size,
      closable: true,
      onClose: handleClose,
    },
    {
      default: () =>
        h("div", { class: "flex items-center gap-x-2" }, [
          h("div", {
            class: "w-4 h-4 rounded",
            style: `background-color: ${color};`,
          }),
          value,
        ]),
    }
  );
};
</script>
