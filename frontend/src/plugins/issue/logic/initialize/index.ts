import { computed, ref, Ref, watch } from "vue";
import { useRoute } from "vue-router";
import { Issue, IssueCreate, IssueType } from "@/types";
import { pushNotification, useIssueStore } from "@/store";
import { idFromSlug } from "@/utils";
import { defaultTemplate, templateForType } from "@/plugins";
import { BuildNewIssueContext } from "../types";
import { maybeBuildTenantDeployIssue } from "./tenant";
import { maybeBuildGhostIssue } from "./ghost";
import { buildNewStandardIssue } from "./standard";

export function useInitializeIssue(issueSlug: Ref<string>) {
  const issueStore = useIssueStore();
  const create = computed(() => issueSlug.value.toLowerCase() == "new");
  const route = useRoute();
  const issue = ref<Issue | IssueCreate | undefined>();
  const template = computed(() => {
    // Find proper IssueTemplate from route.query.template
    const issueType = route.query.template as IssueType;
    if (issueType) {
      const tpl = templateForType(issueType);
      if (tpl) {
        return tpl;
      }
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: `Unknown template '${issueType}'.`,
        description: "Fallback to the default template",
      });
    }

    // fallback
    return defaultTemplate();
  });

  // initialize or re-initialize issue when issueSlug changes
  watch(
    issueSlug,
    async () => {
      issue.value = undefined;
      if (create.value) {
        issue.value = await buildNewIssue({ template, route });
      } else {
        issue.value = issueStore.getIssueById(idFromSlug(issueSlug.value));
      }
    },
    { immediate: true }
  );

  return { create, issue };
}

const buildNewIssue = async (
  context: BuildNewIssueContext
): Promise<IssueCreate> => {
  const tenant = await maybeBuildTenantDeployIssue(context);
  if (tenant) {
    return tenant;
  }

  const ghost = await maybeBuildGhostIssue(context);
  if (ghost) {
    return ghost;
  }

  return buildNewStandardIssue(context);
};
