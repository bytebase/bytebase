<template>
  <BBModal
    :title="$t('schema-editor.actions.create-schema')"
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
        {{ $t("common.create") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { DatabaseId, UNKNOWN_ID } from "@/types";
import { useSchemaEditorStore, useNotificationStore } from "@/store";
import { SchemaMetadata } from "@/types/proto/store/database";
import { convertSchemaMetadataToSchema } from "@/types/schemaEditor/atomType";

const schemaNameFieldRegexp = /^\S+$/;

interface LocalState {
  schemaName: string;
}

const props = defineProps({
  databaseId: {
    type: Number as PropType<DatabaseId>,
    default: UNKNOWN_ID,
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

  const schema = convertSchemaMetadataToSchema(SchemaMetadata.fromPartial({}));
  schema.name = state.schemaName;
  schema.status = "created";
  databaseSchema.schemaList.push(schema);
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
