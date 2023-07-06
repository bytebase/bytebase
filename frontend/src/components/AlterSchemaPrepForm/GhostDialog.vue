<template>
  <BBDialog
    ref="dialog"
    :title="$t('issue.migration-mode.title')"
    :positive-text="$t('common.next')"
    :on-before-positive-click="checkFeature"
    data-label="bb-migration-mode-dialog"
  >
    <div class="w-[28rem] space-y-4 pl-8 pr-2 pb-4">
      <div class="flex items-start space-x-2">
        <input
          v-model="state.mode"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent mt-0.5"
          value="normal"
        />
        <div @click="state.mode = 'normal'">
          <div class="textlabel">
            {{ $t("issue.migration-mode.normal.title") }}
          </div>
          <div class="textinfolabel mt-1">
            <i18n-t tag="p" keypath="issue.migration-mode.normal.description" />
          </div>
        </div>
      </div>
      <div class="flex items-start space-x-2">
        <input
          v-model="state.mode"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent mt-0.5"
          value="online"
        />
        <div @click="state.mode = 'online'">
          <div class="textlabel flex items-center space-x-2">
            <span>{{ $t("issue.migration-mode.online.title") }}</span>
            <FeatureBadge feature="bb.feature.online-migration" />
            <BBBetaBadge />
          </div>
          <div class="textinfolabel mt-1">
            <i18n-t tag="p" keypath="issue.migration-mode.online.description">
              <template #link>
                <LearnMoreLink
                  url="https://www.bytebase.com/docs/change-database/online-schema-migration-for-mysql"
                />
              </template>
            </i18n-t>
          </div>
        </div>
      </div>
    </div>
  </BBDialog>

  <FeatureModal
    feature="bb.feature.online-migration"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue";
import { BBDialog } from "@/bbkit";
import LearnMoreLink from "../LearnMoreLink.vue";
import { featureToRef } from "@/store";

type Mode = "normal" | "online";

type LocalState = {
  mode: Mode;
  showFeatureModal: boolean;
};

const dialog = ref<InstanceType<typeof BBDialog> | null>(null);

const state = reactive<LocalState>({
  mode: "normal",
  showFeatureModal: false,
});

const hasGhostFeature = featureToRef("bb.feature.online-migration");

const checkFeature = (): boolean => {
  if (state.mode === "normal") {
    // Don't block anything when selecting normal migration
    return true;
  }

  if (!hasGhostFeature.value) {
    state.showFeatureModal = true;
    return false;
  }
  return true;
};

const open = (): Promise<{ result: boolean; mode: Mode }> => {
  state.mode = "normal"; // reset state

  return dialog.value!.open().then(
    (result) => {
      return {
        result,
        mode: state.mode,
      };
    },
    (_error) => {
      return {
        result: false,
        mode: "normal",
      };
    }
  );
};

defineExpose({ open });
</script>
