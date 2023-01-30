<template>
  <div class="space-y-2 w-full pl-4 pr-2">
    <div class="flex flex-row justify-between">
      <div class="outline-title group toplevel flex">
        🎈 {{ $t("common.quickstart") }}
      </div>
      <button class="btn-icon" @click.prevent="hideQuickstart">
        <heroicons-solid:x class="w-4 h-4" />
      </button>
    </div>
    <nav class="flex justify-center py-2" aria-label="Progress">
      <ol class="space-y-4 w-full">
        <li v-for="(intro, index) in introList" :key="index">
          <!-- Complete Task -->
          <!-- use <router-link> if intro.link is not empty -->
          <!-- use <span> otherwise -->
          <component
            :is="intro.link ? 'router-link' : 'span'"
            :to="intro.link"
            class="group cursor-pointer"
            @click="intro.click"
          >
            <span class="flex items-start">
              <span
                class="flex-shrink-0 relative h-5 w-5 flex items-center justify-center"
              >
                <template v-if="intro.done.value">
                  <!-- Heroicon name: solid/check-circle -->
                  <heroicons-solid:check-circle
                    class="h-full w-full text-success group-hover:text-success-hover"
                  />
                </template>
                <template v-else-if="isTaskActive(index)">
                  <span
                    class="absolute h-4 w-4 rounded-full bg-blue-200"
                  ></span>
                  <span
                    class="relative block w-2 h-2 bg-info rounded-full"
                  ></span>
                </template>
                <template v-else>
                  <div
                    class="h-2 w-2 bg-gray-300 rounded-full group-hover:bg-gray-400"
                  ></div>
                </template>
              </span>
              <span
                class="ml-2 text-sm font-medium text-control-light group-hover:text-control-light-hover"
                :class="intro.done.value ? 'line-through' : ''"
                >{{ intro.name }}</span
              >
            </span>
          </component>
        </li>
      </ol>
    </nav>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, Ref } from "vue";
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

export default defineComponent({
  name: "QuickStart",
  setup() {
    const uiStateStore = useUIStateStore();
    const { t } = useI18n();
    const kbarHandler = useKBarHandler();

    const currentUser = useCurrentUser();

    const introList: IntroItem[] = [
      {
        name: computed(() => t("quick-start.view-an-issue")),
        link: "/issue/101",
        done: computed(() => uiStateStore.getIntroStateByKey("issue.visit")),
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

    introList.push({
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
    });

    const isTaskActive = (index: number): boolean => {
      for (let i = index - 1; i >= 0; i--) {
        if (!introList[i].done.value) {
          return false;
        }
      }
      return !introList[index].done.value;
    };

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

    return {
      introList,
      isTaskActive,
      hideQuickstart,
    };
  },
});
</script>
