<template>
  <div>
    <div class="mb-4 font-medium">
      {{ $t("settings.general.workspace.database-change-mode.description") }}
      <LearnMoreLink
        url="https://docs.bytebase.com/administration/mode?source=console"
        class="ml-1 text-sm"
      />
      <div class="text-control-placeholder text-sm">
        {{
          $t(
            "settings.general.workspace.database-change-mode.can-be-changed-later"
          )
        }}
      </div>
    </div>
    <div class="w-full flex flex-col gap-8">
      <NRadioGroup
        :disabled="disabled"
        :value="mode"
        @update:value="$emit('update:mode', $event)"
      >
        <NSpace vertical size="large">
          <NRadio :value="DatabaseChangeMode.PIPELINE">
            <div class="flex flex-col gap-1">
              <div class="font-medium">
                {{
                  $t(
                    "settings.general.workspace.database-change-mode.issue-mode.self"
                  )
                }}
              </div>
              <div>
                {{
                  $t(
                    "settings.general.workspace.database-change-mode.issue-mode.description"
                  )
                }}
              </div>
            </div>
          </NRadio>
          <NRadio :value="DatabaseChangeMode.EDITOR">
            <div class="flex flex-col gap-1">
              <div class="font-medium">
                {{
                  $t(
                    "settings.general.workspace.database-change-mode.sql-editor-mode.self"
                  )
                }}
              </div>
              <div>
                {{
                  $t(
                    "settings.general.workspace.database-change-mode.sql-editor-mode.description"
                  )
                }}
              </div>
            </div>
          </NRadio>
        </NSpace>
      </NRadioGroup>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NRadio, NRadioGroup, NSpace } from "naive-ui";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";

withDefaults(
  defineProps<{
    mode: DatabaseChangeMode;
    disabled?: boolean;
  }>(),
  { disabled: false }
);

defineEmits<{
  (event: "update:mode", mode: DatabaseChangeMode): void;
}>();
</script>
