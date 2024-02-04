<template>
  <ContextMenuButton
    :preference-key="`bb-branch-merge-action`"
    :action-list="actions"
    default-action-key="REBASE"
    size="medium"
    @click="
      $emit(
        'perform-action',
        ($event as ContextMenuButtonAction<PostMergeAction>).params
      )
    "
  />
</template>

<script setup lang="ts">
import { ButtonProps } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  ContextMenuButton,
  type ContextMenuButtonAction,
} from "@/components/v2";
import { PostMergeAction } from "./types";

const props = defineProps<{
  buttonProps?: ButtonProps;
}>();

defineEmits<{
  (event: "perform-action", post: PostMergeAction): void;
}>();

const { t } = useI18n();
const actions = computed((): ContextMenuButtonAction<PostMergeAction>[] => {
  return [
    {
      key: "NOOP",
      text: t("branch.merge-rebase.merge-branch"),
      params: "NOOP",
      props: props.buttonProps,
    },
    {
      key: "REBASE",
      text: t("branch.merge-rebase.post-merge-action.rebase"),
      params: "REBASE",
      props: props.buttonProps,
    },
    {
      key: "DELETE",
      text: t("branch.merge-rebase.post-merge-action.delete"),
      params: "DELETE",
      props: props.buttonProps,
    },
  ];
});
</script>
