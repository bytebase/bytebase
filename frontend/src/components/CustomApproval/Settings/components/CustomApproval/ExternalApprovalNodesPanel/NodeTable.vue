<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="nodeList"
    :row-clickable="false"
    :show-placeholder="true"
    row-key="id"
    v-bind="$attrs"
  >
    <template #item="{ item: node }: { item: ExternalApprovalSetting_Node }">
      <div class="bb-grid-cell">
        {{ node.id }}
      </div>
      <div class="bb-grid-cell">
        {{ node.title }}
      </div>
      <div class="bb-grid-cell">
        {{ node.endpoint }}
      </div>
      <div class="bb-grid-cell gap-x-2">
        <NButton size="small" @click="editOrViewNode(node)">
          {{ allowAdmin ? $t("common.edit") : $t("common.view") }}
        </NButton>
        <SpinnerButton
          size="small"
          :tooltip="
            $t('custom-approval.approval-flow.external-approval.delete')
          "
          :disabled="!allowAdmin"
          :on-confirm="() => deleteNode(node)"
        >
          {{ $t("common.delete") }}
        </SpinnerButton>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { cloneDeep, pullAt } from "lodash-es";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, type BBGridColumn } from "@/bbkit";
import { pushNotification, useSettingV1Store } from "@/store";
import {
  ExternalApprovalSetting,
  ExternalApprovalSetting_Node,
} from "@/types/proto/v1/setting_service";
import { SpinnerButton } from "../../common";
import { useCustomApprovalContext } from "../context";

const { t } = useI18n();
const context = useCustomApprovalContext();
const settingStore = useSettingV1Store();
const {
  hasFeature,
  showFeatureModal,
  allowAdmin,
  externalApprovalNodeContext,
} = context;

const settingValue = computed(() => {
  const setting = settingStore.getSettingByName(
    "bb.workspace.approval.external"
  );
  return (
    setting?.value?.externalApprovalSettingValue ??
    ExternalApprovalSetting.fromJSON({})
  );
});

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    { title: t("common.id"), width: "minmax(16rem, auto)" },
    { title: t("common.name"), width: "minmax(auto, 16rem)" },
    {
      title: t("custom-approval.approval-flow.external-approval.endpoint"),
      width: "1fr",
    },
    {
      title: t("common.operations"),
      width: "10rem",
    },
  ];

  return columns;
});

const nodeList = computed(() => {
  return settingValue.value.nodes;
});

const editOrViewNode = (node: ExternalApprovalSetting_Node) => {
  if (!hasFeature.value) {
    showFeatureModal.value = true;
    return;
  }
  externalApprovalNodeContext.value = {
    mode: "EDIT",
    node,
  };
};

const deleteNode = async (node: ExternalApprovalSetting_Node) => {
  try {
    const settingValue = ExternalApprovalSetting.fromJSON({});
    try {
      const setting = await settingStore.fetchSettingByName(
        "bb.workspace.approval.external",
        true /* silent */
      );
      if (
        setting &&
        setting.value &&
        setting.value.externalApprovalSettingValue
      ) {
        Object.assign(settingValue, setting.value.externalApprovalSettingValue);
      }
    } catch {
      // nothing
    }

    const settingValuePatch = cloneDeep(settingValue);
    const index = settingValuePatch.nodes.findIndex((n) => n.id === node.id);
    pullAt(settingValuePatch.nodes, index);
    await settingStore.upsertSetting({
      name: "bb.workspace.approval.external",
      value: {
        externalApprovalSettingValue: settingValuePatch,
      },
    });

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.deleted"),
    });
  } catch {
    // nothing, exception has been handled already
  }
};
</script>
