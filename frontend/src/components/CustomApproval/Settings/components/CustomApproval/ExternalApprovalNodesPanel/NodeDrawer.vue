<template>
  <Drawer :show="show" @close="cancel">
    <DrawerContent :title="title" class="w-[32rem]">
      <div class="flex flex-col gap-y-4">
        <div class="space-y-1">
          <label class="block font-medium text-control space-x-1">
            <RequiredStar />
            {{ $t("common.id") }}
          </label>
          <div>
            <NInput
              v-model:value="state.node.id"
              :disabled="!allowAdmin || mode === 'EDIT'"
            />
          </div>
        </div>

        <div class="space-y-1">
          <label class="block font-medium text-control space-x-1">
            <RequiredStar />
            {{ $t("common.name") }}
          </label>
          <div>
            <NInput v-model:value="state.node.title" :disabled="!allowAdmin" />
          </div>
        </div>

        <div class="space-y-1">
          <div class="flex space-x-2">
            <label class="block font-medium text-control space-x-1">
              <RequiredStar />
              {{
                $t("custom-approval.approval-flow.external-approval.endpoint")
              }}
            </label>
            <LearnMoreLink
              url="https://www.bytebase.com/docs/api/external-approval/?source=console"
              class="text-sm"
            />
          </div>
          <div>
            <NInput
              v-model:value="state.node.endpoint"
              type="textarea"
              :autosize="{
                minRows: 1,
                maxRows: 5,
              }"
              :disabled="!allowAdmin"
              :allow-input="trimEndpoint"
              placeholder="https://approval.acme.com/api/..."
              style="width: 100%"
            />
          </div>
        </div>
      </div>

      <div
        v-if="state.loading"
        v-zindexable="{ enabled: true }"
        class="absolute inset-0 bg-white/50 flex flex-col items-center justify-center"
      >
        <BBSpin />
      </div>

      <template #footer>
        <div class="flex items-center justify-end gap-x-3">
          <NButton @click="cancel">
            {{ $t("common.cancel") }}
          </NButton>

          <NButton
            v-if="allowAdmin"
            type="primary"
            :disabled="!isValid"
            tag="div"
            @click="handleUpsert"
          >
            {{ mode === "CREATE" ? $t("common.create") : $t("common.update") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { cloneDeep } from "lodash-es";
import { NButton, NInput } from "naive-ui";
import { zindexable as vZindexable } from "vdirs";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent } from "@/components/v2";
import { useSettingV1Store } from "@/store";
import { ExternalApprovalSetting_Node } from "@/types/proto/store/setting";
import { ExternalApprovalSetting } from "@/types/proto/v1/setting_service";
import { RequiredStar } from "../../common";
import { useCustomApprovalContext } from "../context";

type LocalState = {
  node: ExternalApprovalSetting_Node;
  loading: boolean;
};

const { t } = useI18n();
const settingStore = useSettingV1Store();
const context = useCustomApprovalContext();
const { allowAdmin, externalApprovalNodeContext: nodeContext } = context;
const state = reactive<LocalState>({
  node: ExternalApprovalSetting_Node.fromJSON({}),
  loading: false,
});

const show = computed(() => {
  return !!nodeContext.value;
});

const mode = computed(() => {
  return nodeContext.value?.mode ?? "CREATE";
});

const title = computed(() => {
  if (nodeContext.value) {
    if (!allowAdmin.value) {
      return t("custom-approval.approval-flow.external-approval.view-node");
    }
    const { mode } = nodeContext.value;
    if (mode === "CREATE") {
      return t("custom-approval.approval-flow.external-approval.create-node");
    }
    if (mode === "EDIT") {
      return t("custom-approval.approval-flow.external-approval.edit-node");
    }
  }
  return "";
});

const cancel = () => {
  nodeContext.value = undefined;
};

const trimEndpoint = (value: string) => {
  return (
    !value.startsWith(" ") && !value.endsWith(" ") && !value.includes("\n")
  );
};

const isValid = computed(() => {
  const errors: string[] = [];
  const { node } = state;
  if (!node.id) {
    return false;
  }
  if (!node.title) {
    return false;
  }
  if (!node.endpoint) {
    return false;
  }
  return errors;
});

const handleUpsert = async () => {
  try {
    state.loading = true;

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
    const { node } = state;
    const index = settingValuePatch.nodes.findIndex((n) => n.id === node.id);
    if (index >= 0) {
      settingValuePatch.nodes[index] = node;
    } else {
      settingValuePatch.nodes.push(node);
    }

    await settingStore.upsertSetting({
      name: "bb.workspace.approval.external",
      value: {
        externalApprovalSettingValue: settingValuePatch,
      },
    });

    cancel();
  } finally {
    state.loading = false;
  }
};

watch(
  () => nodeContext.value?.node,
  (node) => {
    if (node) {
      state.node = cloneDeep(node);
    }
  },
  { immediate: true }
);
</script>
