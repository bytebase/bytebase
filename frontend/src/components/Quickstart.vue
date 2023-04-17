<template>
  <div
    class="py-2 px-4 w-full flex-shrink-0 border-t border-block-border hidden lg:block"
  >
    <p
      class="text-sm font-medium text-gray-900 flex items-center justify-between"
    >
      <span>🎈 {{ $t("common.quickstart") }}</span>

      <button class="btn-icon" @click.prevent="hideQuickstart">
        <heroicons-solid:x class="w-4 h-4" />
      </button>
    </p>
    <div class="mt-2" aria-hidden="true">
      <div class="overflow-hidden rounded-full bg-gray-200">
        <div
          class="h-2 rounded-full bg-indigo-600"
          :style="{ width: percent }"
        />
      </div>
      <div
        class="mt-2 hidden grid-cols-4 text-sm font-medium text-gray-600 sm:grid"
        :style="{
          gridTemplateColumns: `repeat(${introList.length}, minmax(0, 1fr))`,
        }"
      >
        <component
          :is="intro.link ? 'router-link' : 'div'"
          v-for="(intro, index) in introList"
          :key="index"
          :to="intro.link"
          active-class=""
          exact-active-class=""
          class="group cursor-pointer flex items-center gap-x-1 text-sm font-medium"
          :class="[
            index === 0 && 'justify-start',
            index > 0 && index < introList.length - 1 && 'justify-center',
            index === introList.length - 1 && 'justify-end',
            isTaskActive(index)
              ? 'text-indigo-600'
              : 'text-control-light group-hover:text-control-light-hover',
            unref(intro.done) && 'line-through',
          ]"
          @click="intro.click"
        >
          <span
            class="relative h-5 w-5 inline-flex items-center justify-center"
          >
            <template v-if="intro.done.value">
              <!-- Heroicon name: solid/check-circle -->
              <heroicons-solid:check-circle
                class="h-full w-full text-success group-hover:text-success-hover"
              />
            </template>
            <template v-else-if="isTaskActive(index)">
              <span class="absolute h-4 w-4 rounded-full bg-blue-200"></span>
              <span class="relative block w-2 h-2 bg-info rounded-full"></span>
            </template>
            <template v-else>
              <div
                class="h-2 w-2 bg-gray-300 rounded-full group-hover:bg-gray-400"
              ></div>
            </template>
          </span>
          <span class="inline-flex">
            {{ unref(intro.name) }}
          </span>
        </component>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, unref, Ref } from "vue";
import { useI18n } from "vue-i18n";
import { hasWorkspacePermission } from "../utils";
import { useKBarHandler, useKBarEventOnce } from "@bytebase/vue-kbar";
import { pushNotification, useCurrentUser, useUIStateStore } from "@/store";

type IntroItem = {
  name: string | Ref<string>;
  link: string;
  done: Ref<boolean>;
  click?: () => void;
};

const uiStateStore = useUIStateStore();
const { t } = useI18n();
const kbarHandler = useKBarHandler();

const currentUser = useCurrentUser();

const introList = computed(() => {
  const introList: IntroItem[] = [
    {
      name: computed(() =>
        t("quick-start.use-kbar", {
          shortcut: `${navigator.platform.match(/mac/i) ? "cmd" : "ctrl"}-k`,
        })
      ),
      link: "",
      click: () => {
        kbarHandler.value.show();
      },
      done: computed(() => uiStateStore.getIntroStateByKey("kbar.open")),
    },
    {
      name: computed(() => t("quick-start.view-an-issue")),
      link: "/issue/101",
      done: computed(() => uiStateStore.getIntroStateByKey("issue.visit")),
    },
    {
      name: computed(() => t("quick-start.query-data")),
      link: "/sql-editor/sheet/sample-sheet-101",
      done: computed(() => uiStateStore.getIntroStateByKey("data.query")),
    },
    {
      name: computed(() => t("quick-start.visit-project")),
      link: "/project",
      done: computed(() => uiStateStore.getIntroStateByKey("project.visit")),
    },
  ];

  if (
    hasWorkspacePermission(
      "bb.permission.workspace.manage-environment",
      currentUser.value.role
    )
  ) {
    introList.push({
      name: computed(() => t("quick-start.visit-environment")),
      link: "/environment",
      done: computed(() =>
        uiStateStore.getIntroStateByKey("environment.visit")
      ),
    });
  }

  if (
    hasWorkspacePermission(
      "bb.permission.workspace.manage-instance",
      currentUser.value.role
    )
  ) {
    introList.push({
      name: computed(() => t("quick-start.visit-instance")),
      link: "/instance",
      done: computed(() => uiStateStore.getIntroStateByKey("instance.visit")),
    });
  }

  introList.push({
    name: computed(() => t("quick-start.visit-database")),
    link: "/db",
    done: computed(() => uiStateStore.getIntroStateByKey("database.visit")),
  });

  if (
    hasWorkspacePermission(
      "bb.permission.workspace.manage-member",
      currentUser.value.role
    )
  ) {
    introList.push({
      name: computed(() => t("quick-start.add-a-member")),
      link: "/setting/member",
      done: computed(() =>
        uiStateStore.getIntroStateByKey("member.addOrInvite")
      ),
    });
  }

  return introList;
});

const currentStep = computed(() => {
  let i = 0;
  const list = introList.value;
  while (i < list.length && list[i].done.value) {
    i++;
  }
  return i;
});

const isTaskActive = (index: number): boolean => {
  for (let i = index - 1; i >= 0; i--) {
    if (!introList.value[i].done.value) {
      return false;
    }
  }
  return !introList.value[index].done.value;
};

const progress = computed(() => {
  return {
    current: currentStep.value,
    total: introList.value.length,
  };
});

const percent = computed(() => {
  const { current, total } = progress.value;
  if (current === 0) {
    return "3rem";
  }
  if (current === total - 1) {
    return "calc(100% - 3rem)";
  }

  const offset = 0.5;
  const unit = 100 / total;
  const percent = Math.min((current + offset) * unit, 100);
  return `${percent}%`;
});

const hideQuickstart = () => {
  uiStateStore
    .saveIntroStateByKey({
      key: "hidden",
      newState: true,
    })
    .then(() => {
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("quick-start.notice.title"),
        description: t("quick-start.notice.desc"),
        manualHide: true,
      });
    });
};

useKBarEventOnce("open", () => {
  // once kbar is open, mark the quickstart as done
  uiStateStore.saveIntroStateByKey({
    key: "kbar.open",
    newState: true,
  });
});
</script>
