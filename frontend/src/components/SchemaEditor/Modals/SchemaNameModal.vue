<template>
  <BBModal
    :title="
      isCreatingSchema
        ? $t('schema-editor.actions.create-schema')
        : $t('schema-editor.actions.rename')
    "
    class="shadow-inner outline outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-72">
      <p>{{ $t("schema-editor.schema.name") }}</p>
      <BBTextField
        class="my-2 w-full"
        :required="true"
        :focus-on-mount="true"
        :value="state.schemaName"
        @input="handleSchemaNameChange"
      />
    </div>
    <div class="w-full flex items-center justify-end mt-2 space-x-3 pr-1 pb-1">
      <button type="button" class="btn-normal" @click="dismissModal">
        {{ $t("common.cancel") }}
      </button>
      <button class="btn-primary" @click="handleConfirmButtonClick">
        {{ isCreatingSchema ? $t("common.create") : $t("common.save") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  useSchemaEditorStore,
  useNotificationStore,
  generateUniqueTabId,
} from "@/store";
import { SchemaEditorTabType, UNKNOWN_ID } from "@/types";
import { SchemaMetadata } from "@/types/proto/store/database";
import { convertSchemaMetadataToSchema } from "@/types/schemaEditor/atomType";

const schemaNameFieldRegexp = /^\S+$/;

interface LocalState {
  schemaName: string;
}

const props = defineProps({
  databaseId: {
    type: String,
    default: String(UNKNOWN_ID),
  },
  schemaId: {
    type: String,
    default: undefined,
  },
});

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const editorStore = useSchemaEditorStore();
const notificationStore = useNotificationStore();
const state = reactive<LocalState>({
  schemaName: "",
});

const isCreatingSchema = computed(() => {
  return props.schemaId === undefined;
});

onMounted(() => {
  if (props.schemaId === undefined) {
    return;
  }

  const schema = editorStore.getSchema(props.databaseId, props.schemaId);
  if (schema) {
    state.schemaName = schema.name;
  }
});

const handleSchemaNameChange = (event: Event) => {
  state.schemaName = (event.target as HTMLInputElement).value;
};

const handleConfirmButtonClick = async () => {
  if (!schemaNameFieldRegexp.test(state.schemaName)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.invalid-schema-name"),
    });
    return;
  }

  const databaseId = props.databaseId;
  const databaseSchema = editorStore.databaseSchemaById.get(databaseId);
  if (!databaseSchema) {
    return;
  }
  const schemaNameList =
    databaseSchema.schemaList.map((schema) => schema.name) || [];
  if (schemaNameList.includes(state.schemaName)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.duplicated-schema-name"),
    });
    return;
  }

  if (isCreatingSchema.value) {
    const schema = convertSchemaMetadataToSchema(
      SchemaMetadata.fromPartial({})
    );
    schema.name = state.schemaName;
    schema.status = "created";
    databaseSchema.schemaList.push(schema);
    editorStore.addTab({
      id: generateUniqueTabId(),
      type: SchemaEditorTabType.TabForDatabase,
      databaseId: databaseId,
      selectedSchemaId: schema.id,
    });
  } else {
    const schema = editorStore.getSchema(
      props.databaseId,
      props.schemaId ?? ""
    );
    if (schema) {
      schema.name = state.schemaName;
    }
  }
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
