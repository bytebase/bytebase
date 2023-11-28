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
          <NButton class="relative">
            <template #icon>
              <UploadIcon class="w-4 h-4 text-control-light" />
            </template>
            <template #default>
              {{ $t("issue.upload-sql") }}
              <input
                id="sql-file-input"
                type="file"
                accept=".sql,.txt,application/sql,text/plain"
                class="opacity-0 absolute inset-0"
                @change="handleUploadFile"
              />
            </template>
          </NButton>
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
import { UploadIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, onMounted, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ActionConfirmModal from "@/components/SchemaEditorV1/Modals/ActionConfirmModal.vue";
import { useDBGroupStore, useNotificationStore } from "@/store";
import { ComposedDatabaseGroup, ComposedSchemaGroup } from "@/types";
import { generateDatabaseGroupIssueRoute } from "@/utils/databaseGroup/issue";
import { MonacoEditor } from "../MonacoEditor";

const MAX_UPLOAD_FILE_SIZE_MB = 1;

interface LocalState {
  editStatement: string;
  showActionConfirmModal: boolean;
}

const props = defineProps({
  databaseGroup: {
    type: Object as PropType<ComposedDatabaseGroup>,
    required: true,
  },
  issueType: {
    type: String as PropType<
      "bb.issue.database.schema.update" | "bb.issue.database.data.update"
    >,
    required: true,
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
});
const notificationStore = useNotificationStore();

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

onMounted(async () => {
  const schemaGroupList =
    await dbGroupStore.getOrFetchSchemaGroupListByDBGroupName(
      props.databaseGroup.name
    );
  // Initial statement with schema group list;
  state.editStatement = generateReferenceStatement(schemaGroupList);
});

const generateReferenceStatement = (schemaGroupList: ComposedSchemaGroup[]) => {
  const statementList: string[] = [];
  for (const schemaGroup of schemaGroupList) {
    if (props.issueType === "bb.issue.database.schema.update") {
      statementList.push(
        `ALTER TABLE ${schemaGroup.tablePlaceholder} ADD COLUMN <<column>> <<datatype>>;`
      );
    } else {
      statementList.push(
        `UPDATE ${schemaGroup.tablePlaceholder} SET <<column>> = <<value>> WHERE <<condition>>;`
      );
    }
  }
  return statementList.join("\n\n");
};

const dismissModal = () => {
  if (allowPreviewIssue.value) {
    state.showActionConfirmModal = true;
  } else {
    emit("close");
  }
};

const handleUploadFile = (e: Event) => {
  const target = e.target as HTMLInputElement;
  const file = (target.files || [])[0];
  const cleanup = () => {
    // Note that once selected a file, selecting the same file again will not
    // trigger <input type="file">'s change event.
    // So we need to do some cleanup stuff here.
    target.files = null;
    target.value = "";
  };

  if (!file) {
    return cleanup();
  }
  if (file.size > MAX_UPLOAD_FILE_SIZE_MB * 1024 * 1024) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("issue.upload-sql-file-max-size-exceeded", {
        size: `${MAX_UPLOAD_FILE_SIZE_MB}MB`,
      }),
    });
    return cleanup();
  }
  const fr = new FileReader();
  fr.onload = () => {
    const sql = fr.result as string;
    state.editStatement = sql;
  };
  fr.onerror = () => {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "WARN",
      title: `Read file error`,
      description: String(fr.error),
    });
    return;
  };
  fr.readAsText(file);

  cleanup();
};

const handlePreviewIssue = async () => {
  const issueRoute = generateDatabaseGroupIssueRoute(
    props.issueType,
    props.databaseGroup,
    state.editStatement
  );
  router.push(issueRoute);
};
</script>
