<template>
  <div class="flex flex-row items-center gap-x-4 mt-2">
    <NRadio
      v-for="sshType in SshTypes"
      :key="sshType"
      :value="sshType"
      :checked="state.type === sshType"
      @update:checked="handleSelectType(sshType, $event)"
    >
      <span class="textlabel">{{ getSshTypeLabel(sshType) }}</span>
    </NRadio>
  </div>

  <template v-if="state.type === 'TUNNEL' || state.type === 'TUNNEL+PK'">
    <div
      class="sm:col-span-1 sm:col-start-1 mt-4 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4"
    >
      <div class="sm:col-span-3 sm:col-start-1">
        <label for="sshHost" class="textlabel block">
          {{ $t("data-source.ssh.host") }}
        </label>
        <NInput
          v-model:value="state.value.sshHost"
          class="mt-1 w-full"
          :placeholder="''"
        />
      </div>

      <div class="sm:col-span-1">
        <label for="sshPort" class="textlabel block">
          {{ $t("data-source.ssh.port") }}
        </label>
        <NInput
          v-model:value="state.value.sshPort"
          class="mt-1 w-full"
          :placeholder="''"
          :allow-input="onlyAllowNumber"
        />
      </div>
    </div>

    <div
      class="mt-2 grid grid-cols-1 gap-y-2 gap-x-4 border-none sm:grid-cols-3"
    >
      <div class="mt-2 sm:col-span-1 sm:col-start-1">
        <label for="sshUser" class="textlabel block">
          {{ $t("data-source.ssh.user") }}
        </label>
        <NInput
          v-model:value="state.value.sshUser"
          class="mt-1 w-full"
          :placeholder="''"
        />
      </div>
      <div class="mt-2 sm:col-span-1 sm:col-start-1">
        <label for="sshPassword" class="textlabel block">
          {{ $t("data-source.ssh.password") }}
        </label>
        <NInput
          v-model:value="state.value.sshPassword"
          class="mt-1 w-full"
          :placeholder="$t('instance.password-write-only')"
        />
      </div>
    </div>
    <div
      v-if="state.type === 'TUNNEL+PK'"
      class="mt-4 sm:col-span-3 sm:col-start-1"
    >
      <div class="mt-2 sm:col-span-1 sm:col-start-1 flex flex-col gap-y-1">
        <label for="sshPrivateKey" class="textlabel block">
          {{ $t("data-source.ssh.ssh-key") }}
        </label>
        <DroppableTextarea
          v-model:value="state.value.sshPrivateKey"
          :resizable="false"
          class="w-full h-24 whitespace-pre-wrap"
        />
      </div>
    </div>
  </template>

  <FeatureModal
    feature="bb.feature.instance-ssh-connection"
    :open="state.showFeatureModal"
    :instance="instance"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { NInput, NRadio } from "naive-ui";
import { PropType, reactive, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import DroppableTextarea from "@/components/misc/DroppableTextarea.vue";
import { useSubscriptionV1Store } from "@/store";
import { Instance } from "@/types/proto/v1/instance_service";
import { onlyAllowNumber } from "./common";

const SshTypes = ["NONE", "TUNNEL", "TUNNEL+PK"] as const;

type SshType = "NONE" | "TUNNEL" | "TUNNEL+PK";

type WithSshOptions = {
  sshHost?: string;
  sshPort?: string;
  sshUser?: string;
  sshPassword?: string;
  sshPrivateKey?: string;
};

type LocalState = {
  type: SshType;
  value: WithSshOptions;
  tab: "TUNNEL" | "TUNNEL+PK";
  showFeatureModal: boolean;
};

const props = defineProps({
  value: {
    type: Object as PropType<WithSshOptions>,
    required: true,
  },
  instance: {
    type: Object as PropType<Instance>,
    default: undefined,
  },
});

const emit = defineEmits<{
  (e: "change", value: WithSshOptions): void;
}>();

const { t } = useI18n();

const state = reactive<LocalState>({
  type: guessSshType(props.value),
  value: {
    sshHost: props.value.sshHost,
    sshPort: props.value.sshPort,
    sshUser: props.value.sshUser,
    sshPassword: props.value.sshPassword,
    sshPrivateKey: props.value.sshPrivateKey,
  },
  tab: "TUNNEL",
  showFeatureModal: false,
});

const hasSSHConnectionFeature = computed(() => {
  return useSubscriptionV1Store().hasInstanceFeature(
    "bb.feature.instance-ssh-connection",
    props.instance
  );
});

const handleSelectType = (type: SshType, checked: boolean) => {
  if (!checked) return;

  if (!hasSSHConnectionFeature.value) {
    if (type !== "NONE") {
      state.type = "NONE";
      state.showFeatureModal = true;
      return;
    }
  }

  state.type = type;
};

// Sync the latest version to local state when props.value changed.
watch(
  () => props.value,
  (newValue) => {
    state.type = guessSshType(newValue);
    state.value = {
      sshHost: props.value.sshHost,
      sshPort: props.value.sshPort,
      sshUser: props.value.sshUser,
      sshPassword: props.value.sshPassword,
      sshPrivateKey: props.value.sshPrivateKey,
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
      state.value.sshHost = "";
      state.value.sshPort = "";
      state.value.sshUser = "";
      state.value.sshPassword = "";
      state.value.sshPrivateKey = "";
    } else if (type === "TUNNEL") {
      state.value.sshPrivateKey = "";
      state.tab = "TUNNEL";
    }
  }
);

function getSshTypeLabel(type: SshType): string {
  if (type === "TUNNEL") {
    return t("data-source.ssh-type.tunnel");
  }
  if (type === "TUNNEL+PK") {
    return t("data-source.ssh-type.tunnel-and-private-key");
  }
  return t("data-source.ssh-type.none");
}

function guessSshType(value: WithSshOptions): SshType {
  if (value.sshPort) {
    if (value.sshPrivateKey) return "TUNNEL+PK";
    return "TUNNEL";
  }
  return "NONE";
}
</script>
