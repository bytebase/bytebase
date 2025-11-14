<template>
  <BBModal
    :title="
      mode === 'create'
        ? $t('schema-editor.actions.create-view')
        : $t('schema-editor.actions.rename')
    "
    class="shadow-inner outline-solid outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-72">
      <p>{{ $t("common.name") }}</p>
      <NInput
        ref="inputRef"
        v-model:value="state.viewName"
        class="my-2"
        :autofocus="true"
      />
    </div>
    <div class="w-full flex items-center justify-end mt-2 gap-x-2">
      <NButton @click="dismissModal">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton type="primary" @click="handleConfirmButtonClick">
        {{ mode === "create" ? $t("common.create") : $t("common.save") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import type { InputInst } from "naive-ui";
import { NButton, NInput } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
import { useNotificationStore } from "@/store";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { ViewMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";

// View name must start with a non-space character, end with a non-space character.
const viewNameFieldRegexp = /^\S\S*\S?$/;

interface LocalState {
  viewName: string;
}

const props = defineProps<{
  database: ComposedDatabase;
  metadata: DatabaseMetadata;
  schema: SchemaMetadata;
  view?: ViewMetadata;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const { events, addTab, markEditStatus } = useSchemaEditorContext();
const inputRef = ref<InputInst>();
const notificationStore = useNotificationStore();
const mode = computed(() => {
  return props.view ? "edit" : "create";
});
const state = reactive<LocalState>({
  viewName: props.view?.name ?? "",
});

const handleConfirmButtonClick = async () => {
  if (!viewNameFieldRegexp.test(state.viewName)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.invalid-view-name"),
    });
    return;
  }
  const { schema } = props;
  const existed = schema.views.find((view) => view.name === state.viewName);
  if (existed) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.duplicated-view-name"),
    });
    return;
  }

  if (!props.view) {
    const view = create(ViewMetadataSchema, {
      name: state.viewName,
      definition: "",
    });
    schema.views.push(view);
    markEditStatus(
      props.database,
      {
        schema,
        view,
      },
      "created"
    );

    addTab({
      type: "view",
      database: props.database,
      metadata: {
        database: props.metadata,
        schema: props.schema,
        view,
      },
    });

    events.emit("rebuild-tree", {
      openFirstChild: false,
    });
  } else {
    const { view } = props;
    view.name = state.viewName;
    events.emit("rebuild-edit-status", {
      resets: ["tree"],
    });
  }
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};

onMounted(() => {
  inputRef.value?.focus();
});
</script>
