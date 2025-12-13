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

    <!-- General JSON Config (always visible) -->
    <div class="flex flex-col gap-y-3">
      <div class="flex items-center gap-x-2">
        <span class="text-sm text-control-light">
          {{ $t("settings.mcp.setup.add-to-config") }}
        </span>
        <CopyButton :content="generalConfig" />
      </div>
      <NInput
        type="textarea"
        :value="generalConfig"
        readonly
        :autosize="{ minRows: 5, maxRows: 8 }"
        class="font-mono text-sm"
      />
    </div>

    <!-- CLI Quick Commands Tabs -->
    <div class="flex flex-col gap-y-2">
      <NTabs type="line" animated>
        <!-- Claude Code -->
        <NTabPane
          v-for="tab in tabs"
          :key="tab.id"
          :name="tab.id"
          :tab="tab.title"
        >
          <div class="flex flex-col gap-y-2 pt-3">
            <div class="flex items-center gap-x-2">
              <span class="text-sm text-control-light">
                {{ $t("settings.mcp.setup.run-command") }}
              </span>
              <CopyButton :content="tab.content" />
            </div>
            <NInput
              :value="tab.content"
              readonly
              class="font-mono text-sm"
              v-bind="tab.inputProps"
            />
          </div>
        </NTabPane>
      </NTabs>
      <p class="text-sm text-control-light">
        {{ $t("settings.mcp.setup.oauth-note") }}
      </p>
    </div>

    <!-- Your First Prompt -->
    <div class="flex flex-col gap-y-1">
      <span class="text-sm text-control-light">
        {{ $t("settings.mcp.first-prompt.title") }}
      </span>
      <div class="flex items-center gap-x-2 text-sm text-control-light">
        {{ $t("settings.mcp.first-prompt.description") }}
        <CopyButton :content="$t('settings.mcp.first-prompt.example')" />
      </div>
      <code class="mt-1 text-sm bg-gray-100 px-3 py-2 rounded">
        {{ $t("settings.mcp.first-prompt.example") }}
      </code>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { type InputProps, NInput, NTabPane, NTabs } from "naive-ui";
import { computed } from "vue";
import { CopyButton, MissingExternalURLAttention } from "@/components/v2";
import { useActuatorV1Store } from "@/store";

const actuatorStore = useActuatorV1Store();

const hasExternalUrl = computed(() => {
  return !actuatorStore.needConfigureExternalUrl;
});

const mcpEndpointUrl = computed(() => {
  const url = actuatorStore.serverInfo?.externalUrl ?? "";
  if (!hasExternalUrl.value || !url) {
    return "https://your-bytebase-url.com/mcp";
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
}));

const vscodeConfig = computed(() => {
  const config = {
    "mcp.servers": {
      bytebase: {
        type: "http",
        url: mcpEndpointUrl.value,
      },
    },
  };
  return JSON.stringify(config, null, 2);
});

const tabs = computed(
  (): {
    id: string;
    title: string;
    content: string;
    inputProps?: InputProps;
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
        content: vscodeConfig.value,
        inputProps: {
          type: "textarea",
          autosize: { minRows: 5, maxRows: 8 },
        },
      },
    ];
  }
);
</script>
