import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useCurrentUserV1, useProjectV1ListByCurrentUser } from "@/store";
import { DEFAULT_PROJECT_ID } from "@/types";
import { isMemberOfProjectV1, projectV1Slug } from "@/utils";

export const useProjectActions = () => {
  const { t } = useI18n();
  const router = useRouter();
  const me = useCurrentUserV1();
  const { projectList } = useProjectV1ListByCurrentUser();

  const accessibleProjectList = computed(() => {
    return projectList.value.filter((project) => {
      return (
        project.uid != String(DEFAULT_PROJECT_ID) &&
        // Only show projects that the user is a member of.
        isMemberOfProjectV1(project.iamPolicy, me.value)
      );
    });
  });

  const sortedProjectList = computed(() => {
    return [...accessibleProjectList.value].sort((a, b) =>
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
          router.push({ path: `/project/${projectV1Slug(project)}#overview` });
        },
      })
    );
    return actions;
  });
  useRegisterActions(kbarActions);
};
