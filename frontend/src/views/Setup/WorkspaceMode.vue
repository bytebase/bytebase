<template>
  <div>
    <div class="mb-4 font-medium">
      {{ $t("settings.general.workspace.default-landing-page.self") }}
    </div>
    <div class="w-full flex flex-col gap-8">
      <NRadioGroup
        :disabled="disabled"
        :value="mode"
        :size="'large'"
        @update:value="$emit('update:mode', $event)"
      >
        <NSpace vertical size="large">
          <NRadio :value="DatabaseChangeMode.PIPELINE">
            <div class="flex flex-col gap-1">
              <div class="textinfo">
                {{
                  $t(
                    "settings.general.workspace.default-landing-page.workspace.self"
                  )
                }}
              </div>
              <div class="textinfolabel">
                {{
                  $t(
                    "settings.general.workspace.default-landing-page.workspace.description"
                  )
                }}
              </div>
            </div>
          </NRadio>
          <NRadio :value="DatabaseChangeMode.EDITOR">
            <div class="flex flex-col gap-1">
              <div class="textinfo">
                {{
                  $t(
                    "settings.general.workspace.default-landing-page.sql-editor.self"
                  )
                }}
              </div>
              <div class="textinfolabel">
                {{
                  $t(
                    "settings.general.workspace.default-landing-page.sql-editor.description"
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
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";

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
