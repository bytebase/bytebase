<template>
  <div class="flex flex-col gap-y-2">
    <div v-if="!hasFeature(PlanLimitConfig_Feature.PRE_DEPLOYMENT_SQL_REVIEW)">
      <div class="flex space-x-4 flex-1">
        <NButton size="small" @click.prevent="state.showFeatureModal = true">
          🎈{{ $t("sql-review.unlock-full-feature") }}
        </NButton>
      </div>
    </div>

    <div class="w-full">
      <div
        v-if="sdlState.loading"
        class="h-20 flex flex-col items-center justify-center"
      >
        <BBSpin />
      </div>
      <template v-else-if="sdlState.detail">
        <NTabs v-model:value="state.tab" class="mb-1">
          <NTab name="DIFF" :disabled="!!sdlState.detail.error">
            <div class="flex items-center gap-x-1">
              {{ $t("issue.sdl.schema-change") }}
              <NTooltip :disabled="!!sdlState.detail.error">
                <template #trigger>
                  <heroicons-outline:question-mark-circle class="w-4 h-4" />
                </template>
                <div class="whitespace-nowrap">
                  <span>{{ $t("issue.sdl.left-schema-may-change") }}</span>
                </div>
              </NTooltip>
            </div>
          </NTab>
          <NTab name="STATEMENT" :disabled="!!sdlState.detail.error">
            {{ $t("issue.sdl.generated-ddl-statements") }}
          </NTab>
          <NTab name="SCHEMA">
            {{ $t("issue.sdl.full-schema") }}
          </NTab>
        </NTabs>

        <div class="relative min-h-[6rem]">
          <DiffEditor
            v-if="state.tab === 'DIFF'"
            class="border rounded-[3px] overflow-clip"
            :original="sdlState.detail.previousSDL"
            :modified="sdlState.detail.prettyExpectedSDL"
            :readonly="true"
            :auto-height="{
              alignment: 'modified',
              min: 120,
              max: 480,
            }"
          />
          <MonacoEditor
            v-if="state.tab === 'STATEMENT'"
            class="w-full h-auto border rounded-[3px] overflow-clip"
            data-label="bb-issue-sql-editor"
            :content="sdlState.detail.diffDDL"
            :readonly="true"
            :auto-focus="false"
            :auto-height="{ min: 120, max: 360 }"
          />
          <MonacoEditor
            v-if="state.tab === 'SCHEMA'"
            class="w-full h-auto border rounded-[3px] overflow-clip"
            data-label="bb-issue-sql-editor"
            :content="sdlState.detail.expectedSDL"
            :readonly="true"
            :auto-focus="false"
            :auto-height="{ min: 120, max: 360 }"
            :advices="markers"
          />
        </div>
      </template>
    </div>
  </div>
  <FeatureModal
    :open="state.showFeatureModal"
    :feature="PlanLimitConfig_Feature.PRE_DEPLOYMENT_SQL_REVIEW"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { NTabs, NTab, NTooltip, NButton } from "naive-ui";
import { reactive } from "vue";
import { BBSpin } from "@/bbkit";
import { FeatureModal } from "@/components/FeatureGuard";
import { useIssueContext } from "@/components/IssueV1/logic";
import { DiffEditor, MonacoEditor } from "@/components/MonacoEditor";
import { hasFeature, pushNotification } from "@/store";
import { PlanLimitConfig_Feature } from "@/types/proto/v1/subscription_service";
import { useSQLAdviceMarkers } from "../useSQLAdviceMarkers";
import { useSDLState } from "./useSDLState";

type TabView = "DIFF" | "STATEMENT" | "SCHEMA";

interface LocalState {
  showFeatureModal: boolean;
  tab: TabView;
}

const state = reactive<LocalState>({
  showFeatureModal: false,
  tab: "DIFF",
});

const { state: sdlState, events: sdlEvents } = useSDLState();
const context = useIssueContext();

const { markers } = useSQLAdviceMarkers(context, undefined);

sdlEvents.on("error", ({ message }) => {
  state.tab = "SCHEMA";
  pushNotification({
    module: "bytebase",
    style: "CRITICAL",
    title: message,
  });
});
</script>
