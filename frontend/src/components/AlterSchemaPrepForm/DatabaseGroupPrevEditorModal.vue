<template>
  <BBModal
    :title="title"
    class="w-auto h-auto overflow-auto"
    @close="dismissModal"
  >
    <div
      class="w-192 h-auto flex flex-col justify-start items-start overflow-y-auto gap-y-2"
    >
      <div
        class="w-full h-auto shrink-0 flex flex-row justify-between items-center space-x-4 my-2"
      >
        <div class="textinfolabel">
          {{ $t("database-group.prev-editor.description") }}
          <LearnMoreLink
            url="https://www.bytebase.com/docs/change-database/batch-change/#change-databases-from-database-groups?source=console"
            class="ml-1"
          />
        </div>
        <div class="flex flex-row justify-end items-center shrink-0">
          <SQLUploadButton
            @update:sql="(statement) => (state.editStatement = statement)"
          >
            {{ $t("issue.upload-sql") }}
          </SQLUploadButton>
        </div>
      </div>
      <div class="relative w-full h-96 border rounded overflow-clip">
        <MonacoEditor
          v-model:content="state.editStatement"
          class="w-full min-h-full"
        />
      </div>
    </div>
    <div class="w-full flex flex-row justify-end items-center mt-4">
      <div class="flex justify-end items-center gap-x-3">
        <NCheckbox v-if="isDev()" v-model:checked="state.planOnly">
          {{ $t("issue.sql-review-only") }}
        </NCheckbox>
        <NButton @click="dismissModal">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!allowPreviewIssue"
          @click="handlePreviewIssue"
        >
          {{ $t("schema-editor.preview-issue") }}
        </NButton>
      </div>
    </div>
  </BBModal>

  <!-- Close modal confirm dialog -->
  <ActionConfirmModal
    v-model:show="state.showActionConfirmModal"
    :title="$t('schema-editor.confirm-to-close.title')"
    :description="$t('schema-editor.confirm-to-close.description')"
    @confirm="emit('close')"
  />
</template>

<script lang="ts" setup>
import { NButton, NCheckbox } from "naive-ui";
import type { PropType } from "vue";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { MonacoEditor } from "@/components/MonacoEditor";
import { ActionConfirmModal } from "@/components/SchemaEditorLite";
import SQLUploadButton from "@/components/misc/SQLUploadButton.vue";
import { useDBGroupStore } from "@/store";
import type { ComposedDatabaseGroup } from "@/types";
import { isDev } from "@/utils";
import { generateDatabaseGroupIssueRoute } from "@/utils/databaseGroup/issue";

interface LocalState {
  editStatement: string;
  showActionConfirmModal: boolean;
  // planOnly is used to indicate whether only to create plan.
  planOnly: boolean;
}

const props = defineProps({
  databaseGroupName: {
    type: String,
    required: true,
  },
  issueType: {
    type: String as PropType<
      "bb.issue.database.schema.update" | "bb.issue.database.data.update"
    >,
    required: true,
  },
  planOnly: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  editStatement: "",
  showActionConfirmModal: false,
  planOnly: props.planOnly,
});

const databaseGroup = computed(() => {
  return dbGroupStore.getDBGroupByName(
    props.databaseGroupName
  ) as ComposedDatabaseGroup;
});

const allowPreviewIssue = computed(() => {
  return state.editStatement !== "";
});

const title = computed(() => {
  if (props.issueType === "bb.issue.database.schema.update") {
    return t("database.edit-schema");
  } else {
    return t("database.change-data");
  }
});

const dismissModal = () => {
  if (allowPreviewIssue.value) {
    state.showActionConfirmModal = true;
  } else {
    emit("close");
  }
};

const handlePreviewIssue = async () => {
  const issueRoute = generateDatabaseGroupIssueRoute(
    props.issueType,
    databaseGroup.value,
    state.editStatement,
    state.planOnly
  );
  router.push(issueRoute);
};
</script>
