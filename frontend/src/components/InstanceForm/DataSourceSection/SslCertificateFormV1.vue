<template>
  <div class="mt-2 space-y-3">
    <div class="flex flex-row items-center gap-2">
      <NSwitch
        v-model:value="state.value.verifyTlsCertificate"
        size="small"
        :disabled="disabled"
      />
      <label for="verifyTlsCertificate" class="textlabel block">
        {{ $t("data-source.ssl.verify-certificate") }}
      </label>
      <NTooltip>
        <template #trigger>
          <Info class="w-4 h-4 text-yellow-600" />
        </template>
        {{ $t("data-source.ssl.verify-certificate-tooltip") }}
      </NTooltip>
    </div>

    <NTabs :default-value="'CA'" pane-style="padding-top: 0.25rem">
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
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { Info } from "lucide-vue-next";
import { NTabs, NTabPane, NSwitch, NTooltip } from "naive-ui";
import { computed, reactive, watch } from "vue";
import DroppableTextarea from "@/components/misc/DroppableTextarea.vue";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { DataSource } from "@/types/proto-es/v1/instance_service_pb";

type WithSslOptions = Partial<
  Pick<DataSource, "sslCa" | "sslCert" | "sslKey" | "verifyTlsCertificate">
>;

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
    verifyTlsCertificate: props.value.verifyTlsCertificate ?? false,
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
      verifyTlsCertificate: newValue.verifyTlsCertificate ?? false,
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
