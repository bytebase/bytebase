<template>
  <BBDialog
    ref="dialog"
    :title="$t('issue.migration-mode.title')"
    :positive-text="$t('common.next')"
  >
    <div class="w-[24rem] space-y-4 pl-8 pr-2 pb-4">
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
          <div class="textlabel">
            {{ $t("issue.migration-mode.online.title") }}
          </div>
          <div class="textinfolabel mt-1">
            <i18n-t tag="p" keypath="issue.migration-mode.online.description">
              <template #link>
                <LearnMoreLink
                  url="https://github.com/bytebase/bytebase/blob/main/docs/design/gh-ost-integration.md"
                />
              </template>
            </i18n-t>
          </div>
        </div>
      </div>
    </div>
  </BBDialog>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue";
import { BBDialog } from "@/bbkit";
import LearnMoreLink from "../LearnMoreLink.vue";

type Mode = "normal" | "online";

type LocalState = {
  mode: Mode;
};

const dialog = ref<InstanceType<typeof BBDialog> | null>(null);

const state = reactive<LocalState>({
  mode: "normal",
});

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
