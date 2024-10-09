<template>
  <NPopselect
    v-if="showButton && options.length > 1"
    placement="bottom-start"
    :options="options"
    @update:value="handleSelect"
  >
    <Button v-bind="$attrs" @click="handleClickButton" />
  </NPopselect>
  <NPopover v-else-if="showButton && options.length > 0" placement="bottom-start">
    <template #trigger>
      <Button v-bind="$attrs" @click="handleSelect(options[0].value)" />
    </template>
    <template #default>
      <span>{{ options[0].label }}</span>
    </template>
  </NPopover>
</template>

<script setup lang="ts">
import { NPopover, NPopselect, type SelectOption } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { ChatAction } from "@/plugins/ai";
import { useAIContext } from "@/plugins/ai/logic";
import * as promptUtils from "@/plugins/ai/logic/prompt";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSettingV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { nextAnimationFrame } from "@/utils";
import { useSQLEditorContext } from "../../context";
import Button from "./Button.vue";

defineOptions({
  inheritAttrs: false,
});

const props = defineProps<{
  actions?: ChatAction[];
}>();

const { t } = useI18n();
const tabStore = useSQLEditorTabStore();
const settingV1Store = useSettingV1Store();
const openAIKeySetting = computed(() =>
  settingV1Store.getSettingByName("bb.plugin.openai.key")
);
const openAIKey = computed(
  () => openAIKeySetting.value?.value?.stringValue ?? ""
);

const { showAIPanel } = useSQLEditorContext();
const { instance } = useConnectionOfCurrentSQLEditorTab();
const { events } = useAIContext();

const options = computed(() => {
  const tab = tabStore.currentTab;
  const statement = tab?.selectedStatement || tab?.statement;
  const options: (SelectOption & { value: ChatAction; hide?: boolean })[] = [
    {
      value: "explain-code",
      label: t("plugin.ai.actions.explain-code"),
      hide: !statement,
    },
    {
      value: "find-problems",
      label: t("plugin.ai.actions.find-problems"),
      hide: !statement,
    },
    { value: "chat", label: t("plugin.ai.actions.chat") },
  ];
  return options.filter((opt) => {
    if (opt.hide) return false;
    if (props.actions && !props.actions.includes(opt.value)) return false;
    return true;
  });
});

const showButton = computed(() => {
  return (
    openAIKey.value &&
    !tabStore.isDisconnected &&
    tabStore.currentTab?.mode === "WORKSHEET" &&
    options.value.length > 0
  );
});

const handleSelect = async (action: ChatAction) => {
  showAIPanel.value = true;
  if (action === "explain-code" || action === "find-problems") {
    const tab = tabStore.currentTab;
    const statement = tab?.selectedStatement || tab?.statement;
    if (!statement) return;

    await nextAnimationFrame();
    events.emit("new-conversation");
    await nextAnimationFrame();
    if (action === "explain-code") {
      events.emit("send-chat", {
        content: promptUtils.explainCode(statement, instance.value.engine),
      });
    }
    if (action === "find-problems") {
      events.emit("send-chat", {
        content: promptUtils.findProblems(statement, instance.value.engine),
      });
    }
  }
};

const handleClickButton = () => {
  showAIPanel.value = true;
};
</script>
