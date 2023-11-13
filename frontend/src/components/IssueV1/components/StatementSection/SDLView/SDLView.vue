<template>
  <div class="flex flex-col gap-y-2">
    <div v-if="!hasFeature('bb.feature.sql-review')">
      <div class="flex space-x-4 flex-1">
        <button
          type="button"
          class="btn-small py-0.5 inline-flex items-center text-accent"
          @click.prevent="state.showFeatureModal = true"
        >
          🎈{{ $t("sql-review.unlock-full-feature") }}
        </button>
      </div>
    </div>

    <div class="issue-debug">
      <div>SDLView</div>
      <pre>sdlState: {{ sdlState }}</pre>
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
                  <LearnMoreLink
                    url="https://www.bytebase.com/docs/change-database/state-based-migration/#caveats?source=console"
                    color="light"
                    class="ml-1"
                  />
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

        <DiffEditorV2
          v-if="state.tab === 'DIFF'"
          class="h-[64rem] max-h-full border rounded-md overflow-clip"
          :original="sdlState.detail.previousSDL"
          :modified="sdlState.detail.prettyExpectedSDL"
          :readonly="true"
        />
        <MonacoEditorV2
          v-if="state.tab === 'STATEMENT'"
          class="w-full border h-auto"
          data-label="bb-issue-sql-editor"
          :content="sdlState.detail.diffDDL"
          :readonly="true"
          :auto-focus="false"
          :auto-height="{ min: 120, max: 360 }"
        />
        <MonacoEditorV2
          v-if="state.tab === 'SCHEMA'"
          class="w-full border h-auto"
          data-label="bb-issue-sql-editor"
          :content="sdlState.detail.expectedSDL"
          :readonly="true"
          :auto-focus="false"
          :auto-height="{ min: 120, max: 360 }"
          :advices="markers"
        />
      </template>
    </div>
  </div>
  <FeatureModal
    :open="state.showFeatureModal"
    feature="bb.feature.sql-review"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { NTabs, NTab, NTooltip } from "naive-ui";
import { reactive } from "vue";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { DiffEditorV2, MonacoEditorV2 } from "@/components/MonacoEditor";
import { hasFeature, pushNotification } from "@/store";
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

const { markers } = useSQLAdviceMarkers();

sdlEvents.on("error", ({ message }) => {
  state.tab = "SCHEMA";
  pushNotification({
    module: "bytebase",
    style: "CRITICAL",
    title: message,
  });
});
</script>
