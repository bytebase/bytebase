import type { Action } from "@bytebase/vue-kbar";
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useCurrentUserV1, useProjectV1List } from "@/store";
import { DEFAULT_PROJECT_ID } from "@/types";
import { hasProjectPermissionV2 } from "@/utils";

export const useProjectActions = (limit: number) => {
  const { t } = useI18n();
  const router = useRouter();
  const me = useCurrentUserV1();
  const { projectList } = useProjectV1List();

  const accessibleProjectList = computed(() => {
    return projectList.value.filter((project) => {
      return (
        project.uid != String(DEFAULT_PROJECT_ID) &&
        hasProjectPermissionV2(project, me.value, "bb.projects.get")
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
        // here `id` looks like "bb.project.1234"
        id: `bb.project.${project.uid}`,
        section: t("common.projects"),
        name: project.title,
        keywords: ["project", project.key].join(" "),
        data: {
          tags: [project.key],
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
