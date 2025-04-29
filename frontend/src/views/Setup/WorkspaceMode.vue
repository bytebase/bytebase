<template>
  <div class="w-full flex flex-col gap-8">
    <NRadioGroup :value="mode" @update:value="$emit('update:mode', $event)">
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

        <div class="text-control-placeholder text-sm">
          {{
            $t(
              "settings.general.workspace.database-change-mode.can-be-changed-later"
            )
          }}
          <LearnMoreLink
            url="https://www.bytebase.com/docs/administration/mode?source=console"
            class="ml-1 text-sm"
          />
        </div>
      </NSpace>
    </NRadioGroup>
  </div>

  <AuthFooter />
</template>

<script lang="ts" setup>
import { NRadio, NRadioGroup, NSpace } from "naive-ui";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import AuthFooter from "@/views/auth/AuthFooter.vue";

defineProps<{
  mode: DatabaseChangeMode;
}>();

defineEmits<{
  (event: "update:mode", mode: DatabaseChangeMode): void;
}>();
</script>
