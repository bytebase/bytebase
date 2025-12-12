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
    <NAlert v-if="!hasExternalUrl" type="warning" :bordered="false">
      <template #header>
        {{ $t("settings.mcp.setup.external-url-warning") }}
      </template>
      <router-link
        :to="{ name: SETTING_ROUTE_WORKSPACE_GENERAL }"
        class="normal-link"
      >
        {{ $t("settings.mcp.setup.configure-external-url") }}
      </router-link>
    </NAlert>

    <!-- General JSON Config (always visible) -->
    <div class="flex flex-col gap-y-3">
      <div class="flex items-center gap-x-2">
        <span class="text-sm text-control-light">
          {{ $t("settings.mcp.setup.add-to-config") }}
        </span>
        <NTooltip>
          <template #trigger>
            <NButton
              quaternary
              size="small"
              :disabled="!hasExternalUrl"
              @click="copyGeneralConfig"
            >
              <template #icon>
                <CopyIcon class="w-4 h-4" />
              </template>
            </NButton>
          </template>
          {{ $t("common.copy") }}
        </NTooltip>
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
    <div class="flex flex-col gap-y-3">
      <NTabs type="line" animated>
        <!-- Claude Code -->
        <NTabPane name="claude-code" tab="Claude Code">
          <div class="flex flex-col gap-y-3 pt-3">
            <div class="flex items-center gap-x-2">
              <span class="text-sm text-control-light">
                {{ $t("settings.mcp.setup.run-command") }}
              </span>
              <NTooltip>
                <template #trigger>
                  <NButton
                    quaternary
                    size="small"
                    :disabled="!hasExternalUrl"
                    @click="copyCliCommand('claudeCode')"
                  >
                    <template #icon>
                      <CopyIcon class="w-4 h-4" />
                    </template>
                  </NButton>
                </template>
                {{ $t("common.copy") }}
              </NTooltip>
            </div>
            <NInput
              :value="cliCommands.claudeCode"
              readonly
              class="font-mono text-sm"
            />
          </div>
        </NTabPane>

        <!-- Codex -->
        <NTabPane name="codex" tab="Codex">
          <div class="flex flex-col gap-y-3 pt-3">
            <div class="flex items-center gap-x-2">
              <span class="text-sm text-control-light">
                {{ $t("settings.mcp.setup.run-command") }}
              </span>
              <NTooltip>
                <template #trigger>
                  <NButton
                    quaternary
                    size="small"
                    :disabled="!hasExternalUrl"
                    @click="copyCliCommand('codex')"
                  >
                    <template #icon>
                      <CopyIcon class="w-4 h-4" />
                    </template>
                  </NButton>
                </template>
                {{ $t("common.copy") }}
              </NTooltip>
            </div>
            <NInput
              :value="cliCommands.codex"
              readonly
              class="font-mono text-sm"
            />
          </div>
        </NTabPane>

        <!-- Copilot CLI -->
        <NTabPane name="copilot-cli" tab="Copilot CLI">
          <div class="flex flex-col gap-y-3 pt-3">
            <div class="flex items-center gap-x-2">
              <span class="text-sm text-control-light">
                {{ $t("settings.mcp.setup.run-command") }}
              </span>
              <NTooltip>
                <template #trigger>
                  <NButton
                    quaternary
                    size="small"
                    :disabled="!hasExternalUrl"
                    @click="copyCliCommand('copilotCli')"
                  >
                    <template #icon>
                      <CopyIcon class="w-4 h-4" />
                    </template>
                  </NButton>
                </template>
                {{ $t("common.copy") }}
              </NTooltip>
            </div>
            <NInput
              :value="cliCommands.copilotCli"
              readonly
              class="font-mono text-sm"
            />
          </div>
        </NTabPane>

        <!-- Gemini CLI -->
        <NTabPane name="gemini-cli" tab="Gemini CLI">
          <div class="flex flex-col gap-y-3 pt-3">
            <div class="flex items-center gap-x-2">
              <span class="text-sm text-control-light">
                {{ $t("settings.mcp.setup.run-command") }}
              </span>
              <NTooltip>
                <template #trigger>
                  <NButton
                    quaternary
                    size="small"
                    :disabled="!hasExternalUrl"
                    @click="copyCliCommand('geminiCli')"
                  >
                    <template #icon>
                      <CopyIcon class="w-4 h-4" />
                    </template>
                  </NButton>
                </template>
                {{ $t("common.copy") }}
              </NTooltip>
            </div>
            <NInput
              :value="cliCommands.geminiCli"
              readonly
              class="font-mono text-sm"
            />
          </div>
        </NTabPane>

        <!-- VS Code -->
        <NTabPane name="vscode" tab="VS Code">
          <div class="flex flex-col gap-y-3 pt-3">
            <div class="flex items-center gap-x-2">
              <span class="text-sm text-control-light">
                {{ $t("settings.mcp.setup.add-to-config") }}
              </span>
              <NTooltip>
                <template #trigger>
                  <NButton
                    quaternary
                    size="small"
                    :disabled="!hasExternalUrl"
                    @click="copyVSCodeConfig"
                  >
                    <template #icon>
                      <CopyIcon class="w-4 h-4" />
                    </template>
                  </NButton>
                </template>
                {{ $t("common.copy") }}
              </NTooltip>
            </div>
            <NInput
              type="textarea"
              :value="vscodeConfig"
              readonly
              :autosize="{ minRows: 5, maxRows: 8 }"
              class="font-mono text-sm"
            />
          </div>
        </NTabPane>
      </NTabs>
      <p class="text-sm text-control-light">
        {{ $t("settings.mcp.setup.oauth-note") }}
      </p>
    </div>

    <!-- Your First Prompt -->
    <div class="flex flex-col gap-y-2">
      <h3 class="text-base font-medium">
        {{ $t("settings.mcp.first-prompt.title") }}
      </h3>
      <p class="text-sm text-control-light">
        {{ $t("settings.mcp.first-prompt.description") }}
      </p>
      <code class="text-sm bg-gray-100 px-3 py-2 rounded">
        {{ $t("settings.mcp.first-prompt.example") }}
      </code>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import { CopyIcon } from "lucide-vue-next";
import { NAlert, NButton, NInput, NTabPane, NTabs, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { pushNotification, useActuatorV1Store } from "@/store";

const { t } = useI18n();
const actuatorStore = useActuatorV1Store();
const { copy } = useClipboard();

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

const copyGeneralConfig = async () => {
  await copy(generalConfig.value);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.copied"),
  });
};

const copyCliCommand = async (key: keyof typeof cliCommands.value) => {
  await copy(cliCommands.value[key]);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.copied"),
  });
};

const copyVSCodeConfig = async () => {
  await copy(vscodeConfig.value);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.copied"),
  });
};
</script>
