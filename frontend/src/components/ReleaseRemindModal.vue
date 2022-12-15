<template>
  <BBModal
    :title="$t('settings.release.new-version-available')"
    @close="$emit('cancel')"
  >
    <div class="min-w-0 md:min-w-400">
      <div>
        <p class="whitespace-pre-wrap">
          <i18n-t keypath="settings.release.new-version-content">
            <template #tag>
              <a
                class="font-bold underline"
                target="_blank"
                :href="actuatorStore.releaseInfo.latest?.html_url"
              >
                {{ actuatorStore.releaseInfo.latest?.tag_name }}
              </a>
            </template>
          </i18n-t>
        </p>
        <BBCheckbox
          class="mt-3 ml-1"
          :title="$t('settings.release.not-show-till-next-release')"
          :value="actuatorStore.releaseInfo.ignoreRemindModalTillNextRelease"
          @toggle="
            (on) =>
              (actuatorStore.releaseInfo.ignoreRemindModalTillNextRelease = on)
          "
        />
      </div>
      <div class="mt-7 flex justify-end space-x-2">
        <button
          type="button"
          class="btn-normal"
          @click.prevent="$emit('cancel')"
        >
          {{ $t("common.dismiss") }}
        </button>
        <a
          type="button"
          class="btn-primary"
          target="_blank"
          href="https://www.bytebase.com/docs/get-started/install/overview"
          @click="$emit('cancel')"
        >
          {{ $t("common.learn-more") }}
        </a>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { useActuatorStore } from "@/store";

defineEmits(["cancel"]);

const actuatorStore = useActuatorStore();
</script>
