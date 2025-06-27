<template>
  <NTabs :default-value="'CA'" class="mt-2" pane-style="padding-top: 0.25rem">
    <NTabPane
      name="CA"
      :tab="$t('data-source.ssl.ca-cert')"
      display-directive="show"
    >
      <DroppableTextarea
        v-model:value="state.value.sslCa"
        :resizable="false"
        :disabled="disabled"
        class="w-full h-24 whitespace-pre-wrap"
        placeholder="Input or drag and drop YOUR_CA_CERTIFICATE"
      />
    </NTabPane>
    <NTabPane
      v-if="hasSSLKeyField"
      name="KEY"
      :tab="$t('data-source.ssl.client-key')"
      display-directive="show"
    >
      <DroppableTextarea
        v-model:value="state.value.sslKey"
        :resizable="false"
        :disabled="disabled"
        class="w-full h-24 whitespace-pre-wrap"
        placeholder="Input or drag and drop YOUR_CLIENT_KEY"
      />
    </NTabPane>
    <NTabPane
      v-if="hasSSLCertField"
      name="CERT"
      :tab="$t('data-source.ssl.client-cert')"
      display-directive="show"
    >
      <DroppableTextarea
        v-model:value="state.value.sslCert"
        :resizable="false"
        :disabled="disabled"
        class="w-full h-24 whitespace-pre-wrap"
        placeholder="Input or drag and drop YOUR_CLIENT_CERT"
      />
    </NTabPane>
  </NTabs>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { NTabs, NTabPane } from "naive-ui";
import { computed, reactive, watch } from "vue";
import DroppableTextarea from "@/components/misc/DroppableTextarea.vue";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { DataSource } from "@/types/proto/v1/instance_service";

type WithSslOptions = Partial<Pick<DataSource, "sslCa" | "sslCert" | "sslKey">>;

type LocalState = {
  value: WithSslOptions;
};

const props = defineProps<{
  value: WithSslOptions;
  engineType: Engine;
  disabled: boolean;
}>();

const emit = defineEmits<{
  (e: "change", value: WithSslOptions): void;
}>();

const state = reactive<LocalState>({
  value: {
    sslCa: props.value.sslCa,
    sslCert: props.value.sslCert,
    sslKey: props.value.sslKey,
  },
});

const hasSSLKeyField = computed(() => {
  return ![Engine.MSSQL].includes(props.engineType);
});

const hasSSLCertField = computed(() => {
  return ![Engine.MSSQL].includes(props.engineType);
});

// Sync the latest version to local state when props.value changed.
watch(
  () => props.value,
  (newValue) => {
    state.value = {
      sslCa: newValue.sslCa,
      sslCert: newValue.sslCert,
      sslKey: newValue.sslKey,
    };
  }
);

// Emit the latest lo the parent when local value has been edited.
watch(
  () => state.value,
  (localValue) => {
    emit("change", cloneDeep(localValue));
  },
  { deep: true }
);
</script>
