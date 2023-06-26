<template>
  <template v-if="allowArchiveOrRestore">
    <template v-if="instance.state === State.ACTIVE">
      <BBButtonConfirm
        :style="'ARCHIVE'"
        :button-text="$t('instance.archive-this-instance')"
        :ok-text="$t('common.archive')"
        :require-confirm="true"
        :confirm-title="
          $t('instance.archive-instance-instance-name', [instance.title])
        "
        :confirm-description="
          $t(
            'instance.archived-instances-will-not-be-shown-on-the-normal-interface-you-can-still-restore-later-from-the-archive-page'
          )
        "
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
        :style="'RESTORE'"
        :button-text="$t('instance.restore-this-instance')"
        :ok-text="$t('instance.restore')"
        :require-confirm="true"
        :confirm-title="
          $t('instance.restore-instance-instance-name-to-normal-state', [
            instance.title,
          ])
        "
        :confirm-description="''"
        @confirm="archiveOrRestoreInstance(false)"
      />
    </template>
  </template>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { NCheckbox } from "naive-ui";
import { useI18n } from "vue-i18n";

import {
  useCurrentUserV1,
  useInstanceV1Store,
  pushNotification,
} from "@/store";
import { ComposedInstance } from "@/types";
import { State } from "@/types/proto/v1/common";
import { hasWorkspacePermissionV1 } from "@/utils";

const props = defineProps<{
  instance: ComposedInstance;
}>();

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();
const instanceStore = useInstanceV1Store();

const force = ref(false);

const allowArchiveOrRestore = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-instance",
    currentUserV1.value.userRole
  );
});

const archiveOrRestoreInstance = async (archive: boolean) => {
  if (archive) {
    await instanceStore.archiveInstance(props.instance, force.value);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("instance.successfully-archived-instance-updatedinstance-name", [
        props.instance.title,
      ]),
    });
  } else {
    await instanceStore.restoreInstance(props.instance);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("instance.successfully-archived-instance-updatedinstance-name", [
        props.instance.title,
      ]),
    });
  }
};
</script>
