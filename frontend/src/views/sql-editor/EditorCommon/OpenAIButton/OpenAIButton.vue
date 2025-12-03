<template>
  <template v-if="showButton">
    <template v-if="openAIEnabled">
      <NPopselect
        v-if="options.length > 0"
        placement="bottom-end"
        :options="options"
        @update:value="handleSelect"
      >
        <Button v-bind="$attrs" @click="handleClickButton" />
        <template #header>
          <span class="font-semibold">{{ $t("plugin.ai.ai-assistant") }}</span>
        </template>
      </NPopselect>
      <Button v-else v-bind="$attrs" @click="handleClickButton" />
    </template>
    <NPopover v-if="!openAIEnabled" placement="bottom-end">
      <template #trigger>
        <Button v-bind="$attrs" :disabled="true" />
      </template>
      <template #default>
        <div class="flex flex-col">
          <div class="pb-1 border-b">
            <span class="font-semibold">
              {{ $t("plugin.ai.ai-assistant") }}
            </span>
          </div>
          <div class="pt-2 max-w-[20rem] flex flex-col text-gray-500">
            <p>
              {{ $t("plugin.ai.not-configured.self") }}
              <NButton
                v-if="allowConfigure"
                size="small"
                text
                type="primary"
                class="inline-block"
                @click="goConfigure"
              >
                {{ $t("plugin.ai.not-configured.go-to-configure") }}
              </NButton>
              <template v-else>
                {{ $t("plugin.ai.not-configured.contact-admin-to-configure") }}
              </template>
            </p>
          </div>
        </div>
      </template>
    </NPopover>
  </template>
</template>

<script setup lang="ts">
import { NButton, NPopover, NPopselect, type SelectOption } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import type { ChatAction } from "@/plugins/ai";
import { useAIContext } from "@/plugins/ai/logic";
import * as promptUtils from "@/plugins/ai/logic/prompt";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSettingV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { hasWorkspacePermissionV2, nextAnimationFrame } from "@/utils";
import { useSQLEditorContext } from "../../context";
import Button from "./Button.vue";

defineOptions({
  inheritAttrs: false,
});

const props = defineProps<{
  actions?: ChatAction[];
  statement?: string;
}>();

const { t } = useI18n();
const router = useRouter();
const tabStore = useSQLEditorTabStore();
const settingV1Store = useSettingV1Store();

const openAIEnabled = computed(() => {
  const setting = settingV1Store.getSettingByName(Setting_SettingName.AI);
  return setting?.value?.value?.case === "ai"
    ? setting.value.value.value.enabled
    : false;
});

const { showAIPanel } = useSQLEditorContext();
const { instance } = useConnectionOfCurrentSQLEditorTab();
const { events } = useAIContext();

const options = computed(() => {
  const { statement } = props;
  const options: (SelectOption & { value: ChatAction })[] = [
    {
      value: "explain-code",
      label: t("plugin.ai.actions.explain-code"),
      disabled: !statement,
    },
    {
      value: "find-problems",
      label: t("plugin.ai.actions.find-problems"),
      disabled: !statement,
    },
  ];
  return options.filter((opt) => {
    if (props.actions && !props.actions.includes(opt.value)) return false;
    return true;
  });
});

const showButton = computed(() => {
  return !tabStore.isDisconnected && tabStore.currentTab?.mode === "WORKSHEET";
});

const allowConfigure = computed((): boolean => {
  return hasWorkspacePermissionV2("bb.settings.set");
});

const handleSelect = async (action: ChatAction) => {
  // start new chat if AI panel is not open
  // continue current chat otherwise
  const newChat = !showAIPanel.value;

  showAIPanel.value = true;
  if (action === "explain-code" || action === "find-problems") {
    const { statement } = props;
    if (!statement) return;

    await nextAnimationFrame();
    if (action === "explain-code") {
      events.emit("send-chat", {
        content: promptUtils.explainCode(statement, instance.value.engine),
        newChat,
      });
    }
    if (action === "find-problems") {
      events.emit("send-chat", {
        content: promptUtils.findProblems(statement, instance.value.engine),
        newChat,
      });
    }
  }
};

const handleClickButton = () => {
  showAIPanel.value = !showAIPanel.value;
};

const goConfigure = () => {
  router.push({
    name: SETTING_ROUTE_WORKSPACE_GENERAL,
    hash: "#ai-assistant",
  });
};
</script>
