<template>
  <BBOutline
    :id="'project'"
    :title="$t('common.projects')"
    :item-list="outlineItemList"
    :allow-collapse="false"
  />
</template>

<script lang="ts">
import { computed, defineComponent, watchEffect } from "vue";

import { Project, UNKNOWN_ID, RowStatus } from "../types";
import { projectName, projectSlug } from "../utils";
import { BBOutlineItem } from "../bbkit/types";
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useCurrentUser, useProjectStore } from "@/store";

export default defineComponent({
  name: "ProjectListSidePanel",
  setup() {
    const { t } = useI18n();
    const router = useRouter();

    const currentUser = useCurrentUser();
    const projectStore = useProjectStore();

    const prepareProjectList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        projectStore.fetchProjectListByUser({
          userId: currentUser.value.id,
        });
      }
    };

    watchEffect(prepareProjectList);

    const outlineItemList = computed((): BBOutlineItem[] => {
      const projectList = projectStore.getProjectListByUser(
        currentUser.value.id,
        ["NORMAL"] as RowStatus[]
      );
      return projectList
        .map((item: Project): BBOutlineItem => {
          return {
            id: item.id.toString(),
            name: projectName(item),
            link: `/project/${projectSlug(item)}#overview`,
          };
        })
        .sort((a: any, b: any) => {
          return a.name.localeCompare(b.name);
        });
    });

    const kbarActions = computed((): Action[] => {
      const actions = outlineItemList.value.map((proj: any) =>
        defineAction({
          // here `id` looks like "bb.project.1234"
          id: `bb.project.${proj.id}`,
          section: t("common.projects"),
          name: proj.name,
          keywords: "project",
          perform: () => {
            router.push({ path: proj.link });
          },
        })
      );
      return actions;
    });
    useRegisterActions(kbarActions);

    return {
      outlineItemList,
    };
  },
});
</script>
