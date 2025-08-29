<template>
  <div
    class="w-full flex flex-wrap gap-y-1 justify-between sm:items-center px-1 py-1 border-b bg-white"
  >
    <div
      class="action-left gap-x-2 flex overflow-x-auto sm:overflow-x-hidden items-center pl-1"
    >
      <h3>{{ $t("plugin.ai.ai-assistant") }}</h3>
    </div>

    <div
      class="action-right w-full gap-x-1 flex overflow-x-auto sm:overflow-x-hidden sm:justify-between items-center"
    >
      <NSelect
        v-model:value="selectedProviderType"
        :options="providerOptions"
        size="small"
        :disabled="availableProviders.length <= 1"
        :consistent-menu-width="false"
      />
      <div class="flex items-center">
        <NPopover placement="bottom">
          <template #trigger>
            <NButton
              size="small"
              quaternary
              style="--n-padding: 0 5px"
              @click="showHistoryDialog = true"
            >
              <template #icon>
                <ClockIcon class="w-4 h-4" />
              </template>
            </NButton>
          </template>
          <template #default>
            {{ $t("plugin.ai.conversation.view-history-conversations") }}
          </template>
        </NPopover>
        <NPopover placement="bottom">
          <template #trigger>
            <NButton
              size="small"
              quaternary
              style="--n-padding: 0 5px"
              @click="events.emit('new-conversation', { input: '' })"
            >
              <template #icon>
                <PlusIcon class="w-4 h-4" />
              </template>
            </NButton>
          </template>
          <template #default>
            {{ $t("plugin.ai.conversation.new-conversation") }}
          </template>
        </NPopover>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ClockIcon, PlusIcon } from "lucide-vue-next";
import { NButton, NPopover, NSelect } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { AIProvider_Type } from "@/types/proto-es/v1/setting_service_pb";
import { useAIContext } from "../logic";

const { t } = useI18n();
const { events, showHistoryDialog, aiSetting, provider } = useAIContext();

// Get available providers from settings
const availableProviders = computed(() => {
  return aiSetting.value?.providers || [];
});

// Currently selected provider type
const selectedProviderType = computed({
  get: () => provider.value?.type ?? AIProvider_Type.TYPE_UNSPECIFIED,
  set: (type: AIProvider_Type) => {
    const selected = availableProviders.value.find((p) => p.type === type);
    if (selected) {
      provider.value = selected;
    }
  },
});

// Generate options for the select dropdown
const providerOptions = computed(() => {
  return availableProviders.value.map((p) => ({
    label: getProviderLabel(p.type),
    value: p.type,
  }));
});

// Get display label for a provider type
const getProviderLabel = (type: AIProvider_Type): string => {
  switch (type) {
    case AIProvider_Type.OPEN_AI:
      return t("settings.general.workspace.ai-assistant.provider.open_ai");
    case AIProvider_Type.AZURE_OPENAI:
      return t(
        "settings.general.workspace.ai-assistant.provider.azure_open_ai"
      );
    case AIProvider_Type.GEMINI:
      return t("settings.general.workspace.ai-assistant.provider.gemini");
    case AIProvider_Type.CLAUDE:
      return t("settings.general.workspace.ai-assistant.provider.claude");
    default:
      return "";
  }
};
</script>
