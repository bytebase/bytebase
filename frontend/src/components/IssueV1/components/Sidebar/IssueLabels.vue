<template>
  <div class="flex flex-col gap-y-1">
    <div class="flex items-center gap-x-1 textlabel">
      <span>{{ $t("common.labels") }}</span>
    </div>
    <NSelect
      :multiple="true"
      :options="options"
      :disabled="!hasEditPermission"
      :consistent-menu-width="true"
      max-tag-count="responsive"
      :render-label="renderLabel"
      :render-tag="renderTag"
      :value="issueLabels"
      @update:value="onLablesUpdate"
    />
  </div>
</template>

<script setup lang="ts">
import { NCheckbox, NSelect, NTag } from "naive-ui";
import type { SelectOption } from "naive-ui";
import type { SelectBaseOption } from "naive-ui/lib/select/src/interface";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useIssueContext } from "@/components/IssueV1/logic";
import { issueServiceClient } from "@/grpcweb";
import { pushNotification, useCurrentUserV1 } from "@/store";
import { Issue } from "@/types/proto/v1/issue_service";
import { hasProjectPermissionV2 } from "@/utils";

type IsseuLabelOption = SelectOption & {
  value: string;
  color: string;
};

const { isCreating, issue } = useIssueContext();
const { t } = useI18n();
const currentUser = useCurrentUserV1();

const hasEditPermission = computed(() =>
  hasProjectPermissionV2(
    issue.value.projectEntity,
    currentUser.value,
    isCreating.value ? "bb.issues.create" : "bb.issues.update"
  )
);

const issueLabels = computed(() => {
  const pool = new Set(
    issue.value.projectEntity.issueLabels.map((label) => label.value)
  );
  return issue.value.labels.filter((label) => pool.has(label));
});

const options = computed(() => {
  return issue.value.projectEntity.issueLabels.map<IsseuLabelOption>(
    (label) => ({
      label: label.value,
      value: label.value,
      color: label.color,
    })
  );
});

const onLablesUpdate = async (labels: string[]) => {
  if (isCreating.value) {
    issue.value.labels = labels;
  } else {
    const issuePatch = Issue.fromJSON({
      ...issue.value,
      labels,
    });
    const updated = await issueServiceClient.updateIssue({
      issue: issuePatch,
      updateMask: ["labels"],
    });
    Object.assign(issue.value, updated);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
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

const renderTag = (props: {
  option: SelectBaseOption;
  handleClose: () => void;
}) => {
  const { color, value } = props.option as IsseuLabelOption;

  return h(
    NTag,
    {
      closable: true,
      onClose: props.handleClose,
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
