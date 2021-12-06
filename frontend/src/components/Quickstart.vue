<template>
  <div class="space-y-2">
    <div class="flex flex-row justify-between">
      <div class="outline-title toplevel flex">
        {{ $t("common.quickstart") }}
      </div>
      <button class="btn-icon" @click.prevent="hideQuickstart">
        <svg
          class="w-4 h-4"
          fill="currentColor"
          viewBox="0 0 20 20"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            fill-rule="evenodd"
            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
            clip-rule="evenodd"
          ></path>
        </svg>
      </button>
    </div>
    <nav class="flex justify-center" aria-label="Progress">
      <ol class="space-y-4">
        <li v-for="(intro, index) in effectiveList" :key="index">
          <!-- Complete Task -->
          <router-link :to="intro.link" class="group">
            <span class="flex items-start">
              <span
                class="
                  flex-shrink-0
                  relative
                  h-5
                  w-5
                  flex
                  items-center
                  justify-center
                "
              >
                <template v-if="intro.done">
                  <!-- Heroicon name: solid/check-circle -->
                  <svg
                    class="
                      h-full
                      w-full
                      text-success
                      group-hover:text-success-hover
                    "
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                    aria-hidden="true"
                  >
                    <path
                      fill-rule="evenodd"
                      d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                      clip-rule="evenodd"
                    />
                  </svg>
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
                    class="
                      h-2
                      w-2
                      bg-gray-300
                      rounded-full
                      group-hover:bg-gray-400
                    "
                  ></div>
                </template>
              </span>
              <span
                class="
                  ml-2
                  text-sm
                  font-medium
                  text-control-light
                  group-hover:text-control-light-hover
                "
                :class="intro.done ? 'line-through' : ''"
                >{{ intro.name }}</span
              >
            </span>
          </router-link>
        </li>
      </ol>
    </nav>
  </div>
</template>

<script lang="ts">
import { computed, reactive, ComputedRef } from "vue";
import { useStore } from "vuex";
import { isDBA, isOwner } from "../utils";

type IntroItem = {
  name: string;
  link: string;
  allowDBA: boolean;
  allowDeveloper: boolean;
  done: ComputedRef<boolean>;
};

export default {
  name: "Quickstart",
  components: {},
  props: {},
  emits: [""],
  setup() {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const introList: IntroItem[] = reactive([
      {
        name: "Bookmark an issue",
        link: "/issue/hello-world-101",
        allowDBA: true,
        allowDeveloper: true,
        done: computed(() => {
          return store.getters["uistate/introStateByKey"]("bookmark.create");
        }),
      },
      {
        name: "Add a comment",
        link: "/issue/hello-world-101#activity101",
        allowDBA: true,
        allowDeveloper: true,
        done: computed(() => {
          return store.getters["uistate/introStateByKey"]("comment.create");
        }),
      },
      {
        name: "Visit project",
        link: "/project",
        allowDBA: true,
        allowDeveloper: true,
        done: computed(() => {
          return store.getters["uistate/introStateByKey"]("project.visit");
        }),
      },
      {
        name: "Visit environment",
        link: "/environment",
        allowDBA: true,
        allowDeveloper: false,
        done: computed(() => {
          return store.getters["uistate/introStateByKey"]("environment.visit");
        }),
      },
      {
        name: "Visit instance",
        link: "/instance",
        allowDBA: true,
        allowDeveloper: !store.getters["plan/feature"]("bb.dba-workflow"),
        done: computed(() => {
          return store.getters["uistate/introStateByKey"]("instance.visit");
        }),
      },
      {
        name: "Visit database",
        link: "/db",
        allowDBA: true,
        allowDeveloper: true,
        done: computed(() =>
          store.getters["uistate/introStateByKey"]("database.visit")
        ),
      },
      {
        name: "Add a member",
        link: "/setting/member",
        allowDBA: false,
        allowDeveloper: false,
        done: computed(() =>
          store.getters["uistate/introStateByKey"]("member.addOrInvite")
        ),
      },
    ]);

    const effectiveList = computed(() => {
      if (isOwner(currentUser.value.role)) {
        return introList;
      }
      if (isDBA(currentUser.value.role)) {
        return introList.filter((item) => item.allowDBA);
      }
      return introList.filter((item) => item.allowDeveloper);
    });

    const isTaskActive = (index: number): boolean => {
      for (let i = index - 1; i >= 0; i--) {
        if (!effectiveList.value[i].done) {
          return false;
        }
      }
      return !effectiveList.value[index].done;
    };

    const hideQuickstart = () => {
      store
        .dispatch("uistate/saveIntroStateByKey", {
          key: "hidden",
          newState: true,
        })
        .then(() => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "INFO",
            title: `Quickstart guide dismissed`,
            description:
              "You can still bring it back later from the top right menu",
            manualHide: true,
          });
        });
    };

    return {
      effectiveList,
      isTaskActive,
      hideQuickstart,
    };
  },
};
</script>
