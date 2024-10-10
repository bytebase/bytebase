<template>
  <NPopselect
    v-if="showButton"
    placement="bottom-end"
    :options="options"
    @update:value="handleSelect"
  >
    <Button v-bind="$attrs" @click="handleClickButton" />
    <template #header>
      <span class="font-semibold">{{ $t("plugin.ai.ai-assistant") }}</span>
    </template>
  </NPopselect>
</template>

<script setup lang="ts">
import { NPopselect, type SelectOption } from "naive-ui";
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
import Button from "@/views/sql-editor/EditorCommon/OpenAIButton/Button.vue";
import { useSQLEditorContext } from "@/views/sql-editor/context";

defineOptions({
  inheritAttrs: false,
});

const props = defineProps<{
  code?: string;
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
  const options: (SelectOption & { value: ChatAction })[] = [
    {
      value: "explain-code",
      label: t("plugin.ai.actions.explain-code"),
    },
  ];
  return options;
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
  // start new chat if AI panel is not open
  // continue current chat otherwise
  const newChat = !showAIPanel.value;

  showAIPanel.value = true;
  if (action !== "explain-code") return;
  const { code } = props;
  if (!code) return;

  await nextAnimationFrame();
  events.emit("send-chat", {
    content: promptUtils.explainCode(code, instance.value.engine),
    newChat,
  });
};

const handleClickButton = () => {
  showAIPanel.value = true;
};
</script>
