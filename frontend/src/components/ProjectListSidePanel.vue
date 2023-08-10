<template>
  <BBOutline
    :id="'project'"
    :title="$t('common.projects')"
    :item-list="outlineItemList"
    :allow-collapse="false"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";

import { useProjectV1ListByCurrentUser } from "@/store";
import { projectV1Slug } from "../utils";
import { BBOutlineItem } from "../bbkit/types";

const { t } = useI18n();
const router = useRouter();
const { projectList } = useProjectV1ListByCurrentUser();

const outlineItemList = computed((): BBOutlineItem[] => {
  return projectList.value
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
