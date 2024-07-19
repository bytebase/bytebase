<template>
  <div class="text-sm group">
    <TextOverflowPopover
      v-if="query"
      :content="query.trim()"
      :max-length="100"
      :max-popover-content-length="1000"
      :line-wrap="false"
      :line-break-replacer="' '"
      code-class="relative"
      placement="top"
    >
      <template #default="{ displayContent }">
        <span class="flex-1 flex flex-row justify-start items-center gap-x-1">
          <span class="line-clamp-1">
            {{ displayContent }}
          </span>
          <NButton
            text
            size="tiny"
            class="invisible group-hover:visible"
            @click="copyStatement"
          >
            <template #icon>
              <CopyIcon class="w-3 h-3" />
            </template>
          </NButton>
        </span>
      </template>
      <template #popover-header>
        <div class="absolute bottom-1 right-1">
          <NButton text size="tiny" @click="copyStatement">
            <template #icon>
              <CopyIcon class="w-3 h-3" />
            </template>
          </NButton>
        </div>
      </template>
    </TextOverflowPopover>
    <div v-else class="text-control-placeholder">-</div>
  </div>
</template>

<script setup lang="ts">
import { CopyIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { useI18n } from "vue-i18n";
import TextOverflowPopover from "@/components/misc/TextOverflowPopover.vue";
import { pushNotification } from "@/store";
import { toClipboard } from "@/utils";

const props = defineProps<{
  query: string;
}>();

const { t } = useI18n();

const copyStatement = () => {
  toClipboard(props.query.trim()).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.copied"),
    });
  });
};
</script>
