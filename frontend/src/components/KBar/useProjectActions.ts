import type { Action } from "@bytebase/vue-kbar";
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useProjectV1List } from "@/store";
import { DEFAULT_PROJECT_NAME } from "@/types";
import { extractProjectResourceName, hasProjectPermissionV2 } from "@/utils";

export const useProjectActions = (limit: number) => {
  const { t } = useI18n();
  const router = useRouter();
  const { projectList } = useProjectV1List();

  const accessibleProjectList = computed(() => {
    return projectList.value.filter((project) => {
      return (
        project.name !== DEFAULT_PROJECT_NAME &&
        hasProjectPermissionV2(project, "bb.projects.get")
      );
    });
  });

  const sortedProjectList = computed(() => {
    return [...accessibleProjectList.value.slice(0, limit)].sort((a, b) =>
      a.title.localeCompare(b.title)
    );
  });
  const kbarActions = computed((): Action[] => {
    const actions = sortedProjectList.value.map((project) =>
      defineAction({
        // here `id` looks like "bb.project.projects/cms"
        id: `bb.project.${project.name}`,
        section: t("common.projects"),
        name: project.title,
        keywords: ["project", extractProjectResourceName(project.name)].join(
          " "
        ),
        data: {
          tags: [extractProjectResourceName(project.name)],
        },
        perform: () => {
          router.push({ path: `/${project.name}` });
        },
      })
    );
    return actions;
  });
  useRegisterActions(kbarActions);
};
