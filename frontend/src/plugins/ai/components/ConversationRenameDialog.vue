<template>
  <BBModal
    :title="$t('plugin.ai.conversation.rename')"
    @close="$emit('cancel')"
  >
    <div class="w-[18rem] flex flex-col items-start gap-y-2">
      <div class="textlabel">{{ $t("common.name") }}</div>
      <div class="w-full">
        <NInput
          v-model:value="state.name"
          class="w-full"
          @keypress.enter="handleRename"
        />
      </div>
      <div
        class="w-full flex items-center justify-end gap-x-2 mt-4 pt-2 border-t"
      >
        <NButton @click="$emit('cancel')">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton type="primary" @click="handleRename">
          {{ $t("common.update") }}
        </NButton>
      </div>
      <div
        v-if="state.loading"
        class="absolute inset-0 bg-white/50 flex flex-col items-center justify-center rounded-lg"
      >
        <BBSpin />
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import { useI18n } from "vue-i18n";
import { NButton, NInput } from "naive-ui";
import { head } from "lodash-es";

import type { Conversation } from "../types";
import { BBModal } from "@/bbkit";
import { useConversationStore } from "../store";

type LocalState = {
  name: string;
  loading: boolean;
};

const props = defineProps<{
  conversation: Conversation;
}>();

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "updated"): void;
}>();

const { t } = useI18n();

const state = reactive<LocalState>({
  name:
    props.conversation.name ||
    head(props.conversation.messageList)?.content ||
    t("plugin.ai.conversation.untitled"),
  loading: false,
});
const store = useConversationStore();

const handleRename = async () => {
  const { conversation } = props;
  const { name } = state;
  conversation.name = name;
  state.loading = true;
  await store.updateConversation(conversation);
  emit("updated");
};
</script>
