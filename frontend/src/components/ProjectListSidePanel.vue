<template>
  <BBOutline
    :id="'project'"
    :title="$t('common.projects')"
    :item-list="outlineItemList"
    :allow-collapse="false"
    :outline-item-class="'pt-0.5 pb-0.5'"
  />
</template>

<script lang="ts" setup>
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useCurrentUserV1, useProjectV1ListByCurrentUser } from "@/store";
import { DEFAULT_PROJECT_ID } from "@/types";
import { BBOutlineItem } from "../bbkit/types";
import { isMemberOfProjectV1, projectV1Slug } from "../utils";

const { t } = useI18n();
const router = useRouter();
const currentUserV1 = useCurrentUserV1();
const { projectList } = useProjectV1ListByCurrentUser();

const filteredProjectList = computed(() => {
  const list = projectList.value.filter(
    (project) =>
      project.uid != String(DEFAULT_PROJECT_ID) &&
      // Only show projects that the user is a member of.
      isMemberOfProjectV1(project.iamPolicy, currentUserV1.value)
  );
  return list;
});

const outlineItemList = computed((): BBOutlineItem[] => {
  return filteredProjectList.value
    .map((project): BBOutlineItem => {
      return {
        id: project.uid,
        name: project.title,
        link: `/project/${projectV1Slug(project)}#overview`,
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
</script>
