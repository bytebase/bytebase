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
    @update:value="onLabelsUpdate"
  >
    <template #empty>
      <NEmpty>
        <template #extra>
          <router-link
            v-if="hasPermission"
            :to="{
              name: PROJECT_V1_ROUTE_SETTINGS,
              params: {
                projectId: getProjectName(project.name),
              },
              hash: '#issue-related',
            }"
            class="textinfolabel normal-link flex items-center gap-x-2"
          >
            {{ $t("project.settings.issue-related.labels.configure-labels") }}
            <ExternalLinkIcon class="w-4 h-auto inline-block" />
          </router-link>
        </template>
      </NEmpty>
    </template>
  </NSelect>
</template>

<script lang="ts">
export const getValidIssueLabels = (
  selected: string[],
  issueLabels: Label[]
) => {
  const pool = new Set(issueLabels.map((label) => label.value));
  return selected.filter((label) => pool.has(label));
};
</script>

<script setup lang="ts">
import { ExternalLinkIcon } from "lucide-vue-next";
import type { SelectOption } from "naive-ui";
import { NCheckbox, NEmpty, NSelect, NTag } from "naive-ui";
import type { SelectBaseOption } from "naive-ui/lib/select/src/interface";
import { computed, h } from "vue";
import { PROJECT_V1_ROUTE_SETTINGS } from "@/router/dashboard/projectV1";
import { getProjectName } from "@/store/modules/v1/common";
import type { Label, Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

type IssueLabelOption = SelectOption & {
  value: string;
  color: string;
};

const props = withDefaults(
  defineProps<{
    disabled: boolean;
    selected: string[];
    project: Project;
    size?: "small" | "medium" | "large";
    maxTagCount?: number | "responsive";
  }>(),
  {
    size: "medium",
    maxTagCount: "responsive",
  }
);

const emit = defineEmits<{
  (event: "update:selected", selected: string[]): void;
}>();

const hasPermission = computed(() => {
  return hasProjectPermissionV2(props.project, "bb.projects.update");
});

const issueLabels = computed(() => {
  return getValidIssueLabels(props.selected, props.project.issueLabels);
});

const options = computed(() => {
  return props.project.issueLabels.map<IssueLabelOption>((label) => ({
    label: label.value,
    value: label.value,
    color: label.color,
  }));
});

const onLabelsUpdate = async (labels: string[]) => {
  emit("update:selected", labels);
};

const renderLabel = (option: IssueLabelOption, selected: boolean) => {
  const { color, value } = option;
  return h("div", { class: "flex items-center gap-x-2" }, [
    h(NCheckbox, { checked: selected, size: "small" }),
    h("div", {
      class: "w-4 h-4 rounded-sm cursor-pointer relative",
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
  const { color, value } = option as IssueLabelOption;

  return h(
    NTag,
    {
      size: props.size,
      closable: !props.disabled,
      onClose: handleClose,
    },
    {
      default: () =>
        h("div", { class: "flex items-center gap-x-2" }, [
          h("div", {
            class: "w-4 h-4 rounded-sm",
            style: `background-color: ${color};`,
          }),
          value,
        ]),
    }
  );
};
</script>
