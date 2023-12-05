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
import { reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useNotificationStore, useSchemaEditorV1Store } from "@/store";
import { SchemaMetadata } from "@/types/proto/v1/database_service";
import { convertSchemaMetadataToSchema } from "@/types/v1/schemaEditor";

const schemaNameFieldRegexp = /^\S+$/;

interface LocalState {
  schemaName: string;
}

const props = defineProps<{
  parentName: string;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const schemaEditorV1Store = useSchemaEditorV1Store();
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

  const parentResource = schemaEditorV1Store.resourceMap[
    schemaEditorV1Store.resourceType
  ].get(props.parentName);
  if (!parentResource) {
    throw new Error(
      `Failed to find parent resource ${props.parentName} of type ${schemaEditorV1Store.resourceType}`
    );
  }

  const schema = convertSchemaMetadataToSchema(
    SchemaMetadata.fromPartial({
      name: state.schemaName,
    }),
    "created"
  );
  parentResource.schemaList.push(schema);
  // TODO(steven): Open the schema tab.
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
