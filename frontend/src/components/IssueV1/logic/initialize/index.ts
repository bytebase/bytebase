import { computed, ref, Ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ComposedIssue } from "@/types";
import { experimentalFetchIssueByUID } from "@/store";
import { idFromSlug } from "@/utils";

export function useInitializeIssue(issueSlug: Ref<string>) {
  const isCreating = computed(() => issueSlug.value.toLowerCase() == "new");
  const route = useRoute();
  const router = useRouter();

  const issueCreate = ref<ComposedIssue | undefined>();
  const issueEntity = ref<ComposedIssue | undefined>();
  const issue = computed(() => {
    if (isCreating.value) {
      return issueCreate.value;
    } else {
      const uid = String(idFromSlug(issueSlug.value));
      if (issueEntity.value?.uid === uid) {
        return issueEntity.value;
      }
      return undefined;
    }
  });

  watch(
    [issueSlug, isCreating],
    async ([issueSlug, isCreating]) => {
      try {
        if (isCreating) {
          issueCreate.value = undefined;
          // TODO:
          // - build plan
          // - set default assignee
          // - preview rollout

          // issueCreate.value = await buildNewIssue({ template, route });
          // if (
          //   issueCreate.value.assigneeId === UNKNOWN_ID ||
          //   issueCreate.value.assigneeId === SYSTEM_BOT_ID
          // ) {
          //   // Try to find a default assignee of the first task automatically.
          //   await tryGetDefaultAssignee(issueCreate.value);
          // }
        } else {
          issueEntity.value = undefined;
          const uid = String(idFromSlug(issueSlug));
          const issue = await experimentalFetchIssueByUID(uid);
          issueEntity.value = issue;
        }
      } catch (error) {
        router.push({ name: "error.404" });
        throw error;
      }
    },
    { immediate: true }
  );

  return { isCreating, issue };
}
