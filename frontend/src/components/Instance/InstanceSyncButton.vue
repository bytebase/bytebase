<template>
  <NDropdown
    :trigger="'click'"
    :options="syncInstnceOptions"
    :render-label="renderDropdownLabel"
    :disabled="!allowSyncInstance || disabled || syncingSchema"
    @select="syncSchema"
  >
    <NButton
      icon-placement="right"
      :loading="syncingSchema"
      :quaternary="quaternary"
      :disabled="!allowSyncInstance || disabled"
      :size="size"
      :type="type"
    >
      <template #icon>
        <ChevronDownIcon class="w-4" />
      </template>
      <template v-if="syncingSchema">
        {{ $t("instance.syncing") }}
      </template>
      <template v-else>
        {{ $t("instance.sync.self") }}
      </template>
    </NButton>
  </NDropdown>
</template>

<script lang="tsx" setup>
import { ChevronDownIcon } from "lucide-vue-next";
import type { DropdownOption } from "naive-ui";
import { NButton, NDropdown, NTooltip } from "naive-ui";
import type { ClientError } from "nice-grpc-web";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

withDefaults(
  defineProps<{
    type?: "default" | "primary";
    size?: "small" | "medium";
    disabled?: boolean;
    quaternary?: boolean;
  }>(),
  {
    type: "default",
    size: "medium",
    disabled: false,
    quaternary: false,
  }
);

const emit = defineEmits<{
  (name: "sync-schema", enableFullSync: boolean): void;
}>();

const { t } = useI18n();
const syncingSchema = ref<boolean>(false);

type SyncInstanceOption = "sync-all" | "sync-new";

const renderDropdownLabel = (option: DropdownOption) => {
  if (option.key === "sync-new") {
    return <span>{option.label}</span>;
  }
  return (
    <NTooltip placement="top-start">
      {{
        trigger: () => option.label,
        default: () => (
          <span class="text-sm text-nowrap">
            {t("instance.sync.sync-all-tip")}
          </span>
        ),
      }}
    </NTooltip>
  );
};

const syncInstnceOptions = computed((): DropdownOption[] => {
  return [
    {
      key: "sync-all",
      label: t("instance.sync.sync-all"),
    },
    {
      key: "sync-new",
      label: t("instance.sync.sync-new"),
    },
  ];
});

const allowSyncInstance = computed(() => {
  return hasWorkspacePermissionV2("bb.instances.sync");
});

const syncSchema = async (option: SyncInstanceOption) => {
  try {
    syncingSchema.value = true;
    if (option === "sync-all") {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("instance.sync.sync-all-tip"),
      });
    }
    emit("sync-schema", option === "sync-all" /* enable full sync */);
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("db.failed-to-sync-schema"),
      description: (error as ClientError).details,
    });
  } finally {
    syncingSchema.value = false;
  }
};
</script>
