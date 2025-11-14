<template>
  <div class="mt-2 flex flex-col gap-y-1">
    <div class="flex flex-row items-center gap-x-1">
      <NSwitch
        v-model:value="state.value.verify"
        size="small"
        :disabled="disabled"
      />
      <label class="textlabel block">
        {{ verifyLabel }}
      </label>
      <NTooltip v-if="showTooltip">
        <template #trigger>
          <Info class="w-4 h-4 text-yellow-600" />
        </template>
        {{ $t("data-source.ssl.verify-certificate-tooltip") }}
      </NTooltip>
    </div>

    <NTabs :default-value="'CA'" pane-style="padding-top: 0.25rem">
      <NTabPane name="CA" :tab="caLabel" display-directive="show">
        <DroppableTextarea
          v-model:value="state.value.ca"
          :resizable="false"
          :disabled="disabled"
          class="w-full h-24 whitespace-pre-wrap"
          :placeholder="'Input or drag and drop YOUR_CA_CERTIFICATE'"
        />
      </NTabPane>
      <NTabPane
        v-if="hasSSLKeyField"
        name="KEY"
        :tab="keyLabel"
        display-directive="show"
      >
        <DroppableTextarea
          v-model:value="state.value.key"
          :resizable="false"
          :disabled="disabled"
          class="w-full h-24 whitespace-pre-wrap"
          :placeholder="'Input or drag and drop YOUR_CLIENT_KEY'"
        />
      </NTabPane>
      <NTabPane
        v-if="hasSSLCertField"
        name="CERT"
        :tab="certLabel"
        display-directive="show"
      >
        <DroppableTextarea
          v-model:value="state.value.cert"
          :resizable="false"
          :disabled="disabled"
          class="w-full h-24 whitespace-pre-wrap"
          :placeholder="'Input or drag and drop YOUR_CLIENT_CERT'"
        />
      </NTabPane>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import { Info } from "lucide-vue-next";
import { NSwitch, NTabPane, NTabs, NTooltip } from "naive-ui";
import { computed, reactive, watch } from "vue";
import DroppableTextarea from "@/components/misc/DroppableTextarea.vue";
import { t } from "@/plugins/i18n";
import { Engine } from "@/types/proto-es/v1/common_pb";

type TlsConfig = {
  verify: boolean;
  ca: string;
  cert: string;
  key: string;
};

type LocalState = {
  value: TlsConfig;
};

const props = withDefaults(
  defineProps<{
    verify: boolean;
    ca?: string;
    cert?: string;
    privateKey?: string;
    engineType?: Engine;
    disabled: boolean;
    verifyLabel?: string;
    caLabel?: string;
    certLabel?: string;
    keyLabel?: string;
    showTooltip?: boolean;
    showKeyField?: boolean;
    showCertField?: boolean;
  }>(),
  {
    ca: "",
    cert: "",
    privateKey: "",
    engineType: Engine.ENGINE_UNSPECIFIED,
    showTooltip: true,
    showKeyField: undefined,
    showCertField: undefined,
    caLabel: () => t("data-source.ssl.ca-cert"),
    certLabel: () => t("data-source.ssl.client-cert"),
    keyLabel: () => t("data-source.ssl.client-key"),
    verifyLabel: () => t("data-source.ssl.verify-certificate"),
  }
);

const emit = defineEmits<{
  (e: "update:verify", value: boolean): void;
  (e: "update:ca", value: string): void;
  (e: "update:cert", value: string): void;
  (e: "update:privateKey", value: string): void;
}>();

const state = reactive<LocalState>({
  value: {
    verify: props.verify ?? false,
    ca: props.ca ?? "",
    cert: props.cert ?? "",
    key: props.privateKey ?? "",
  },
});

const hasSSLKeyField = computed(() => {
  if (props.showKeyField !== undefined) {
    return props.showKeyField;
  }
  return ![Engine.MSSQL].includes(props.engineType);
});

const hasSSLCertField = computed(() => {
  if (props.showCertField !== undefined) {
    return props.showCertField;
  }
  return ![Engine.MSSQL].includes(props.engineType);
});

// Sync the latest version to local state when props changed.
watch(
  () => [props.verify, props.ca, props.cert, props.privateKey] as const,
  ([verify, ca, cert, privateKey]) => {
    state.value = {
      verify: verify ?? false,
      ca: ca ?? "",
      cert: cert ?? "",
      key: privateKey ?? "",
    };
  }
);

// Emit changes to parent
watch(
  () => state.value.verify,
  (value) => {
    emit("update:verify", value);
  }
);

watch(
  () => state.value.ca,
  (value) => {
    emit("update:ca", value);
  }
);

watch(
  () => state.value.cert,
  (value) => {
    emit("update:cert", value);
  }
);

watch(
  () => state.value.key,
  (value) => {
    emit("update:privateKey", value);
  }
);
</script>
