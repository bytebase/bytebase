<template>
  <div
    v-if="allowArchiveOrRestore"
    class="t-6 border-t border-block-border flex justify-between items-center pt-4 pb-2 gap-x-2"
  >
    <template v-if="instance.state === State.ACTIVE">
      <BBButtonConfirm
        :type="'ARCHIVE'"
        :button-text="$t('instance.archive-this-instance')"
        :ok-text="$t('common.archive')"
        :require-confirm="true"
        :confirm-title="
          $t('instance.archive-instance-instance-name', [instance.title])
        "
        :confirm-description="
          $t('instance.archived-instances-will-not-be-displayed')
        "
        class="border-none!"
        @confirm="archiveOrRestoreInstance(true)"
      >
        <div class="mt-3">
          <NCheckbox v-model:checked="force">
            <div class="text-sm font-normal text-control-light">
              {{ $t("instance.force-archive-description") }}
            </div>
          </NCheckbox>
        </div>
      </BBButtonConfirm>
    </template>
    <template v-else-if="instance.state === State.DELETED">
      <BBButtonConfirm
        :type="'RESTORE'"
        :button-text="$t('instance.restore-this-instance')"
        :ok-text="$t('instance.restore')"
        :require-confirm="true"
        :confirm-title="
          $t('instance.restore-instance-instance-name-to-normal-state', [
            instance.title,
          ])
        "
        :confirm-description="''"
        class="border-none!"
        @confirm="archiveOrRestoreInstance(false)"
      />
    </template>
    <ResourceHardDeleteButton
      :resource="instance"
      @delete="hardDeleteInstance"
    />
  </div>
</template>

<script setup lang="ts">
import { NCheckbox } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBButtonConfirm } from "@/bbkit";
import ResourceHardDeleteButton from "@/components/v2/Button/ResourceHardDeleteButton.vue";
import { INSTANCE_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { SETTING_ROUTE_WORKSPACE_ARCHIVE } from "@/router/dashboard/workspaceSetting";
import { pushNotification, useInstanceV1Store } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  instance: Instance;
}>();

const { t } = useI18n();
const instanceStore = useInstanceV1Store();
const router = useRouter();

const force = ref(false);

const allowArchiveOrRestore = computed(() => {
  return (
    hasWorkspacePermissionV2("bb.instances.delete") ||
    hasWorkspacePermissionV2("bb.instances.undelete")
  );
});

const archiveOrRestoreInstance = async (archive: boolean) => {
  if (archive) {
    await instanceStore.archiveInstance(props.instance, force.value);
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("instance.successfully-archived-instance", [
        props.instance.title,
      ]),
    });
  } else {
    await instanceStore.restoreInstance(props.instance);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("instance.successfully-restored-instance", [
        props.instance.title,
      ]),
    });
  }

  if (archive) {
    router.replace({
      name: INSTANCE_ROUTE_DASHBOARD,
    });
  }
};

const hardDeleteInstance = async (resource: string) => {
  await instanceStore.deleteInstance(resource);
  router.replace({
    name: SETTING_ROUTE_WORKSPACE_ARCHIVE,
    hash: "#INSTANCE",
  });
};
</script>
