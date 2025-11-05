<template>
  <BBButtonConfirm
    v-if="release.state === State.ACTIVE"
    :type="'ARCHIVE'"
    :button-text="$t('release.archive-this-release')"
    :require-confirm="true"
    :confirm-title="$t('bbkit.confirm-button.sure-to-archive')"
    :confirm-description="$t('bbkit.confirm-button.can-undo')"
    :ok-text="$t('common.confirm')"
    class="border-none!"
    @confirm="archiveOrRestoreRelease(true)"
  />
  <BBButtonConfirm
    v-else-if="release.state === State.DELETED"
    :type="'RESTORE'"
    :button-text="$t('common.restore')"
    :require-confirm="false"
    class="border-none!"
    @confirm="archiveOrRestoreRelease(false)"
  />
</template>

<script setup lang="ts">
import { BBButtonConfirm } from "@/bbkit";
import { useReleaseStore } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import { useReleaseDetailContext } from "./context";

const { release } = useReleaseDetailContext();
const releaseStore = useReleaseStore();

const archiveOrRestoreRelease = async (archive: boolean) => {
  if (archive) {
    await releaseStore.deleteRelease(release.value.name);
  } else {
    await releaseStore.undeleteRelease(release.value.name);
  }
};
</script>
