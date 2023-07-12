import { computed, ref, Ref } from "vue";
import { useRoute } from "vue-router";
import { experimentalFetchIssueByUID } from "@/store";
import { idFromSlug } from "@/utils";
import { computedAsync } from "@vueuse/core";
import { createIssue } from "./create";

export function useInitializeIssue(issueSlug: Ref<string>) {
  const isCreating = computed(() => issueSlug.value.toLowerCase() == "new");
  const route = useRoute();
  const isInitializing = ref(false);

  const issue = computedAsync(
    async () => {
      console.log("call computed async", isCreating.value, issueSlug.value);
      if (isCreating.value) {
        return createIssue(route);
      } else {
        const uid = String(idFromSlug(issueSlug.value));
        return await experimentalFetchIssueByUID(uid);
      }
    },
    undefined,
    isInitializing
  );

  return { isCreating, issue, isInitializing };
}
