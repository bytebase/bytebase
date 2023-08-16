<template>
  <div v-if="!dismissed" class="w-full px-4 py-2 border-t">
    <NInputGroup>
      <NInput
        size="small"
        class="bb-ai-mock-input !cursor-pointer"
        :disabled="false"
        :placeholder="$t('plugin.ai.text-to-sql-disabled-placeholder')"
        @click.prevent.stop="handleClick"
      >
        <template #prefix>
          <heroicons-outline:sparkles class="w-4 h-4 text-accent" />
        </template>
        <template #suffix>
          <NTooltip>
            <template #trigger>
              <NButton
                :quaternary="true"
                type="primary"
                size="small"
                @click.prevent.stop="handleDismiss"
              >
                <heroicons:x-mark />
              </NButton>
            </template>
            <div class="whitespace-nowrap">
              {{ $t("plugin.ai.dont-show-again") }}
            </div>
          </NTooltip>
        </template>
      </NInput>
    </NInputGroup>
  </div>
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import { NInput, NInputGroup, NButton, NTooltip } from "naive-ui";
import { useRouter } from "vue-router";

const router = useRouter();
const dismissed = useLocalStorage<boolean>(
  "bb.plugin.open-ai.dismiss-placeholder",
  false
);

const handleClick = () => {
  router.push({ name: "setting.workspace.general", hash: "#ai-augmentation" });
};

const handleDismiss = () => {
  dismissed.value = true;
};
</script>

<style lang="postcss">
.bb-ai-mock-input .n-input__input-el {
  @apply !ring-0 !cursor-pointer !placeholder-current;
}
</style>
