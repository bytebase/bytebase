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
import { useNotificationStore } from "@/store";
import { SchemaMetadata } from "@/types/proto/store/database";
import { useSchemaDesignerContext } from "../common";

const schemaNameFieldRegexp = /^\S+$/;

interface LocalState {
  schemaName: string;
}

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const { metadata } = useSchemaDesignerContext();
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

  const schema = SchemaMetadata.fromPartial({});
  metadata.value.schemas.push(schema);
  // TODO(steven): Open the schema tab.
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
