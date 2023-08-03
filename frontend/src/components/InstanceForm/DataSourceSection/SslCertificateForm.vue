<template>
  <div class="radio-set-row mt-2">
    <label v-for="type in SslTypes" :key="type" class="radio">
      <input v-model="state.type" type="radio" class="btn" :value="type" />
      <span class="label">
        {{ getSslTypeLabel(type) }}
      </span>
    </label>
  </div>
  <NTabs
    v-if="state.type === 'CA' || state.type === 'CA+KEY+CERT'"
    v-model:value="state.tab"
    class="mt-2"
    pane-style="padding-top: 0.25rem"
  >
    <NTabPane
      name="CA"
      :tab="$t('data-source.ssl.ca-cert')"
      display-directive="show"
    >
      <DroppableTextarea
        v-model:value="state.value.sslCa"
        class="block w-full resize-none whitespace-pre-wrap h-24"
        placeholder="Input or drag and drop YOUR_CA_CERTIFICATE"
      />
    </NTabPane>
    <NTabPane
      v-if="state.type === 'CA+KEY+CERT'"
      name="KEY"
      :tab="$t('data-source.ssl.client-key')"
      display-directive="show"
    >
      <DroppableTextarea
        v-model:value="state.value.sslKey"
        class="block w-full resize-none whitespace-pre-wrap h-24"
        placeholder="Input or drag and drop YOUR_CLIENT_KEY"
      />
    </NTabPane>
    <NTabPane
      v-if="state.type === 'CA+KEY+CERT'"
      name="CERT"
      :tab="$t('data-source.ssl.client-cert')"
      display-directive="show"
    >
      <DroppableTextarea
        v-model:value="state.value.sslCert"
        class="block w-full resize-none whitespace-pre-wrap h-24"
        placeholder="Input or drag and drop YOUR_CLIENT_CERT"
      />
    </NTabPane>
  </NTabs>
</template>

<script lang="ts" setup>
import { PropType, reactive, watch } from "vue";
import { NTabs, NTabPane } from "naive-ui";
import { useI18n } from "vue-i18n";
import { cloneDeep } from "lodash-es";

import { DataSource } from "@/types/proto/v1/instance_service";
import DroppableTextarea from "@/components/misc/DroppableTextarea.vue";

const SslTypes = ["NONE", "CA", "CA+KEY+CERT"] as const;

type SslType = "NONE" | "CA" | "CA+KEY+CERT";

type WithSslOptions = Partial<Pick<DataSource, "sslCa" | "sslCert" | "sslKey">>;

type LocalState = {
  type: SslType;
  value: WithSslOptions;
  tab: "CA" | "KEY" | "CERT";
};

const props = defineProps({
  value: {
    type: Object as PropType<WithSslOptions>,
    required: true,
  },
});

const emit = defineEmits<{
  (e: "change", value: WithSslOptions): void;
}>();

const { t } = useI18n();

const state = reactive<LocalState>({
  type: guessSslType(props.value),
  value: {
    sslCa: props.value.sslCa,
    sslCert: props.value.sslCert,
    sslKey: props.value.sslKey,
  },
  tab: "CA",
});

// Sync the latest version to local state when props.value changed.
watch(
  () => props.value,
  (newValue) => {
    state.type = guessSslType(newValue);
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

watch(
  () => state.type,
  (type) => {
    if (type === "NONE") {
      state.value.sslCa = "";
      state.value.sslCert = "";
      state.value.sslKey = "";
    } else if (type === "CA") {
      state.value.sslCert = "";
      state.value.sslKey = "";
      state.tab = "CA";
    }
  }
);

function getSslTypeLabel(type: SslType): string {
  if (type === "CA") {
    return t("data-source.ssl-type.ca");
  }
  if (type === "CA+KEY+CERT") {
    return t("data-source.ssl-type.ca-and-key-and-cert");
  }
  return t("data-source.ssl-type.none");
}

function guessSslType(value: WithSslOptions): SslType {
  if (value.sslCa) {
    if (value.sslCert && value.sslKey) return "CA+KEY+CERT";
    return "CA";
  }
  return "NONE";
}
</script>
