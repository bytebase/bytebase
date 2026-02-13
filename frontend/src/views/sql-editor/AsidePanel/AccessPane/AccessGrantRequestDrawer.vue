<template>
  <Drawer
    :show="true"
    width="auto"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="$t('sql-editor.request-data-access')"
      :closable="true"
      class="w-[40rem] max-w-[100vw]"
    >
      <div class="flex flex-col gap-y-6">
        <div class="flex flex-col gap-y-2">
          <label class="text-sm font-medium text-control">
            {{ $t("common.databases") }}
            <RequiredStar class="ml-0.5" />
          </label>
          <DatabaseSelect
            :value="form.targets"
            :project-name="editorStore.project"
            :multiple="true"
            @update:value="
              (val) => (form.targets = (val as string[] | undefined) ?? [])
            "
          />
        </div>

        <div class="flex flex-col gap-y-2">
          <label class="text-sm font-medium text-control">
            {{ $t("common.statement") }}
            <RequiredStar class="ml-0.5" />
          </label>
          <MonacoEditor
            class="border rounded-[3px] h-40"
            :content="form.query"
            language="sql"
            :auto-complete-context="autoCompleteContext"
            @update:content="form.query = $event"
          />
        </div>

        <div class="flex flex-col gap-y-2">
          <label class="text-sm font-medium text-control">
            {{ $t("sql-editor.grant-type-unmask") }}
          </label>
          <div class="flex flex-col gap-y-1">
            <NCheckbox v-model:checked="form.unmask">
              {{ $t("sql-editor.access-type-unmask") }}
            </NCheckbox>
          </div>
        </div>

        <div class="flex flex-col gap-y-2">
          <label class="text-sm font-medium text-control">
            {{ $t("common.duration") }}
            <RequiredStar class="ml-0.5" />
          </label>
          <NSelect
            v-model:value="form.duration"
            :options="durationOptions"
          />
          <NDatePicker
            v-if="form.duration === -1"
            v-model:value="form.customExpireTime"
            type="datetime"
            :is-date-disabled="isDateDisabled"
            clearable
          />
        </div>

        <div class="flex flex-col gap-y-2">
          <label class="text-sm font-medium text-control">
            {{ $t("common.reason") }}
          </label>
          <NInput
            v-model:value="form.reason"
            type="textarea"
            :rows="3"
            :placeholder="$t('common.optional')"
          />
        </div>
      </div>

      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!allowSubmit"
            :loading="isRequesting"
            @click="handleSubmit"
          >
            {{ $t("common.submit") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import { NButton, NCheckbox, NDatePicker, NInput, NSelect } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { MonacoEditor } from "@/components/MonacoEditor";
import RequiredStar from "@/components/RequiredStar.vue";
import { DatabaseSelect, Drawer, DrawerContent } from "@/components/v2";
import { accessGrantServiceClientConnect } from "@/connect";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useCurrentUserV1,
  useSQLEditorStore,
  useSQLEditorTabStore,
} from "@/store";
import {
  AccessGrant_Status,
  AccessGrantSchema,
  CreateAccessGrantRequestSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";
import {
  extractDatabaseResourceName,
  extractIssueUID,
  extractProjectResourceName,
} from "@/utils";
import { useSQLEditorContext } from "../../context";

const props = withDefaults(
  defineProps<{
    // Pre-selected database resource names.
    // Format: "instances/{instance}/databases/{database}"
    targets?: string[];
    // Pre-filled SQL statement.
    query?: string;
    // Pre-check the "Unmask" access type checkbox.
    unmask?: boolean;
  }>(),
  {
    targets: undefined,
    query: "",
    unmask: false,
  }
);

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const currentUser = useCurrentUserV1();
const editorStore = useSQLEditorStore();
const tabStore = useSQLEditorTabStore();
const isRequesting = ref(false);
const { asidePanelTab, highlightAccessGrantName } = useSQLEditorContext();

// Default to the currently connected database if no targets provided.
const defaultTargets = computed(() => {
  if (props.targets) return props.targets;
  const database = tabStore.currentTab?.connection?.database;
  return database ? [database] : [];
});

const form = reactive({
  targets: [...defaultTargets.value],
  query: props.query,
  unmask: props.unmask,
  duration: 4,
  customExpireTime: undefined as number | undefined,
  reason: "",
});

const autoCompleteContext = computed(() => {
  const db = form.targets[0];
  if (!db) return undefined;
  const { instance } = extractDatabaseResourceName(db);
  return { instance, database: db };
});

const durationOptions = computed(() => [
  { label: t("sql-editor.duration-hours", { hours: 1 }), value: 1 },
  { label: t("sql-editor.duration-hours", { hours: 4 }), value: 4 },
  { label: t("sql-editor.duration-day", { days: 1 }), value: 24 },
  { label: t("sql-editor.duration-days", { days: 7 }), value: 168 },
  { label: t("common.custom"), value: -1 },
]);

const isDateDisabled = (ts: number) => {
  return ts < Date.now();
};

const allowSubmit = computed(() => {
  if (form.targets.length === 0) return false;
  if (!form.query.trim()) return false;
  if (form.duration === -1 && !form.customExpireTime) return false;
  return true;
});

const handleSubmit = async () => {
  if (isRequesting.value) return;
  isRequesting.value = true;
  try {
    const ttlSeconds =
      form.duration === -1
        ? BigInt(Math.floor((form.customExpireTime! - Date.now()) / 1000))
        : BigInt(form.duration * 3600);

    const accessGrant = create(AccessGrantSchema, {
      creator: `users/${currentUser.value.email}`,
      targets: form.targets,
      query: form.query,
      unmask: form.unmask,
      reason: form.reason,
      expiration: {
        case: "ttl",
        value: create(DurationSchema, { seconds: ttlSeconds }),
      },
    });

    const response = await accessGrantServiceClientConnect.createAccessGrant(
      create(CreateAccessGrantRequestSchema, {
        parent: editorStore.project,
        accessGrant,
      })
    );

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
    });

    if (response.status === AccessGrant_Status.PENDING && response.issue) {
      const route = router.resolve({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(response.issue),
          issueId: extractIssueUID(response.issue),
        },
      });
      window.open(route.fullPath, "_blank");
    } else {
      asidePanelTab.value = "ACCESS";
      highlightAccessGrantName.value = response.name;
    }
    emit("close");
  } finally {
    isRequesting.value = false;
  }
};
</script>
