<template>
  <div class="w-full flex flex-col gap-y-6 pb-4">
    <!-- Header -->
    <div>
      <h2 class="text-lg font-medium">{{ $t("settings.mcp.title") }}</h2>
      <p class="textinfolabel mt-1">
        {{ $t("settings.mcp.description") }}
      </p>
    </div>

    <!-- Warning if external URL not configured -->
    <MissingExternalURLAttention />

    <!-- Authentication Notice -->
    <NAlert type="info" :show-icon="true">
      <template #header>
        {{ $t("settings.mcp.auth.title") }}
      </template>
      {{ $t("settings.mcp.auth.description") }}
    </NAlert>

    <!-- Section 1: General Config -->
    <div class="flex flex-col gap-y-3">
      <div class="flex flex-col gap-y-1">
        <div class="flex items-center gap-x-2">
          <h3 class="text-base font-medium">
            {{ $t("settings.mcp.general-config.title") }}
          </h3>
          <CopyButton :content="generalConfig" />
        </div>
        <p class="text-sm text-control-light">
          {{ $t("settings.mcp.setup.add-to-config") }}
        </p>
      </div>
      <NInput
        type="textarea"
        :value="generalConfig"
        readonly
        :autosize="{ minRows: 5, maxRows: 8 }"
        class="font-mono text-sm"
      />
    </div>

    <!-- Section 2: Quick Start CLI Commands -->
    <div class="flex flex-col gap-y-3">
      <div class="flex flex-col gap-y-1">
        <div class="flex items-center gap-x-2">
          <h3 class="text-base font-medium">
            {{ $t("settings.mcp.quick-start.title") }}
          </h3>
          <CopyButton :content="activeTabContent" />
        </div>
        <p class="text-sm text-control-light">
          {{ $t("settings.mcp.quick-start.description") }}
        </p>
      </div>
      <NTabs v-model:value="activeTab" type="line" animated>
        <NTabPane
          v-for="tab in tabs"
          :key="tab.id"
          :name="tab.id"
          :tab="tab.title"
        >
          <NInput
            :value="tab.content"
            readonly
            class="font-mono text-sm mt-3"
          />
        </NTabPane>
      </NTabs>
    </div>

    <!-- Section 3: Your First Prompt -->
    <div class="flex flex-col gap-y-3">
      <div class="flex flex-col gap-y-1">
        <div class="flex items-center gap-x-2">
          <h3 class="text-base font-medium">
            {{ $t("settings.mcp.first-prompt.title") }}
          </h3>
          <CopyButton :content="$t('settings.mcp.first-prompt.example')" />
        </div>
        <p class="text-sm text-control-light">
          {{ $t("settings.mcp.first-prompt.description") }}
        </p>
      </div>
      <code class="text-sm bg-gray-100 px-3 py-2 rounded">
        {{ $t("settings.mcp.first-prompt.example") }}
      </code>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NAlert, NInput, NTabPane, NTabs } from "naive-ui";
import { computed, ref } from "vue";
import { CopyButton, MissingExternalURLAttention } from "@/components/v2";
import { useActuatorV1Store } from "@/store";

const actuatorStore = useActuatorV1Store();
const activeTab = ref("claude-code");

const hasExternalUrl = computed(() => {
  return !actuatorStore.needConfigureExternalUrl;
});

const mcpEndpointUrl = computed(() => {
  const url = actuatorStore.serverInfo?.externalUrl ?? "";
  if (!hasExternalUrl.value || !url) {
    return "{https://your-bytebase-url.com}/mcp";
  }
  const base = url.replace(/\/$/, "");
  return `${base}/mcp`;
});

const generalConfig = computed(() => {
  const config = {
    mcpServers: {
      bytebase: {
        type: "http",
        url: mcpEndpointUrl.value,
      },
    },
  };
  return JSON.stringify(config, null, 2);
});

const cliCommands = computed(() => ({
  claudeCode: `claude mcp add bytebase --transport http ${mcpEndpointUrl.value}`,
  codex: `codex mcp add --name bytebase --transport http --url ${mcpEndpointUrl.value}`,
  copilotCli: `gh copilot mcp add bytebase --transport http --url ${mcpEndpointUrl.value}`,
  geminiCli: `gemini mcp add --name bytebase --transport http --url ${mcpEndpointUrl.value}`,
  vscode: `code --add-mcp '{"name":"bytebase","type":"http","url":"${mcpEndpointUrl.value}"}'`,
}));

const tabs = computed(
  (): {
    id: string;
    title: string;
    content: string;
  }[] => {
    return [
      {
        id: "claude-code",
        title: "Claude Code",
        content: cliCommands.value.claudeCode,
      },
      {
        id: "codex",
        title: "Codex",
        content: cliCommands.value.codex,
      },
      {
        id: "copilot-cli",
        title: "Copilot CLI",
        content: cliCommands.value.copilotCli,
      },
      {
        id: "gemini-cli",
        title: "Gemini CLI",
        content: cliCommands.value.geminiCli,
      },
      {
        id: "vscode",
        title: "VS Code",
        content: cliCommands.value.vscode,
      },
    ];
  }
);

const activeTabContent = computed(() => {
  const tab = tabs.value.find((t) => t.id === activeTab.value);
  return tab?.content ?? "";
});
</script>
