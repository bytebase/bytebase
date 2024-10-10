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
    { value: "new-chat", label: t("plugin.ai.actions.new-chat") },
  ];
  return options.filter((opt) => {
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
  if (action === "new-chat") {
    events.emit("new-conversation", { input: "" });
  }
};

const handleClickButton = () => {
  showAIPanel.value = true;
};
</script>
