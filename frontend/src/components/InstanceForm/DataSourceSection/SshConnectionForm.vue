<template>
  <div class="flex flex-row items-center gap-x-4 mt-2">
    <NRadio
      v-for="sshType in SshTypes"
      :key="sshType"
      :value="sshType"
      :disabled="disabled"
      :checked="state.type === sshType"
      @update:checked="handleSelectType(sshType, $event)"
    >
      {{ getSshTypeLabel(sshType) }}
    </NRadio>
  </div>

  <template v-if="state.type !== 'NONE'">
    <div
      class="sm:col-span-1 sm:col-start-1 mt-4 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4"
    >
      <div class="sm:col-span-3 sm:col-start-1">
        <label for="sshHost" class="textlabel block">
          {{ $t("data-source.ssh.host") }}
        </label>
        <NInput
          v-model:value="state.value.sshHost"
          class="mt-2 w-full"
          :placeholder="''"
          :disabled="disabled"
        />
      </div>

      <div class="sm:col-span-1">
        <label for="sshPort" class="textlabel block">
          {{ $t("data-source.ssh.port") }}
        </label>
        <NInput
          v-model:value="state.value.sshPort"
          class="mt-2 w-full"
          :placeholder="''"
          :disabled="disabled"
          :allow-input="onlyAllowNumber"
        />
      </div>
    </div>

    <div
      class="mt-2 grid grid-cols-1 gap-y-2 gap-x-4 border-none sm:grid-cols-3"
    >
      <div class="mt-2 sm:col-span-3 sm:col-start-1">
        <label for="sshUser" class="textlabel block">
          {{ $t("data-source.ssh.user") }}
        </label>
        <NInput
          v-model:value="state.value.sshUser"
          class="mt-2 w-full"
          :placeholder="''"
          :disabled="disabled"
        />
      </div>
      <div class="mt-2 sm:col-span-3 sm:col-start-1">
        <label for="sshPassword" class="textlabel block">
          {{ $t("data-source.ssh.password") }}
        </label>
        <NInput
          v-model:value="state.value.sshPassword"
          class="mt-2 w-full"
          :placeholder="$t('instance.password-write-only')"
          :disabled="disabled"
        />
      </div>
    </div>
    <div class="mt-4 sm:col-span-3 sm:col-start-1">
      <div class="mt-2 sm:col-span-1 sm:col-start-1 flex flex-col">
        <label for="sshPrivateKey" class="textlabel block">
          {{ $t("data-source.ssh.ssh-key") }}
          ({{ t("common.optional") }})
        </label>
        <DroppableTextarea
          v-model:value="state.value.sshPrivateKey"
          :resizable="false"
          :disabled="disabled"
          :placeholder="$t('common.write-only')"
          class="w-full h-24 mt-2 whitespace-pre-wrap"
        />
      </div>
    </div>
  </template>
</template>

<script lang="ts" setup>
import DroppableTextarea from "@/components/misc/DroppableTextarea.vue";
import type { Instance } from "@/types/proto/v1/instance_service";
import { onlyAllowNumber } from "@/utils";
import { NInput, NRadio } from "naive-ui";
import { reactive, watch } from "vue";
import { useI18n } from "vue-i18n";

const SshTypes = ["NONE", "TUNNEL+PK"] as const;

type SshType = (typeof SshTypes)[number];

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
};

const props = defineProps<{
  value: WithSshOptions;
  instance?: Instance;
  disabled: boolean;
}>();

const emit = defineEmits<{
  (e: "change", value: WithSshOptions): void;
}>();

const { t } = useI18n();

const state = reactive<LocalState>({
  type: guessSshType(props.value),
  value: {},
});

const handleSelectType = (type: SshType, checked: boolean) => {
  if (!checked) return;

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
  },
  {
    immediate: true,
  }
);

// Emit the latest lo the parent when local value has been edited.
watch(
  () => state.value,
  (localValue) => {
    emit("change", { ...localValue });
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
    }
  }
);

function getSshTypeLabel(type: SshType): string {
  if (type === "TUNNEL+PK") {
    return t("data-source.ssh-type.tunnel-and-private-key");
  }
  return t("data-source.ssh-type.none");
}

function guessSshType(value: WithSshOptions): SshType {
  if (value.sshPort) {
    return "TUNNEL+PK";
  }
  return "NONE";
}
</script>
