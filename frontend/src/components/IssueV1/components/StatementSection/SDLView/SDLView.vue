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

        <CodeDiff
          v-if="state.tab === 'DIFF'"
          :old-string="sdlState.detail.previousSDL"
          :new-string="sdlState.detail.prettyExpectedSDL"
          output-format="side-by-side"
          data-label="bb-change-detail-code-diff-block"
        />
        <MonacoEditor
          v-if="state.tab === 'STATEMENT'"
          ref="editorRef"
          class="w-full border h-auto max-h-[360px]"
          data-label="bb-issue-sql-editor"
          :value="sdlState.detail.diffDDL"
          :readonly="true"
          :auto-focus="false"
          language="sql"
          @ready="handleMonacoEditorReady"
        />
        <MonacoEditor
          v-if="state.tab === 'SCHEMA'"
          ref="editorRef"
          class="w-full border h-auto max-h-[360px]"
          data-label="bb-issue-sql-editor"
          :value="sdlState.detail.expectedSDL"
          :readonly="true"
          :auto-focus="false"
          :advices="markers"
          language="sql"
          @ready="handleMonacoEditorReady"
        />
      </template>
    </div>
  </div>
  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.sql-review"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { NTabs, NTab, NTooltip } from "naive-ui";
import { CodeDiff } from "v-code-diff";
import { reactive, ref } from "vue";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import MonacoEditor from "@/components/MonacoEditor";
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
const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const { state: sdlState, events: sdlEvents } = useSDLState();

const updateEditorHeight = () => {
  const contentHeight =
    editorRef.value?.editorInstance?.getContentHeight() as number;
  editorRef.value?.setEditorContentHeight(contentHeight);
};

const handleMonacoEditorReady = () => {
  updateEditorHeight();
};

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
