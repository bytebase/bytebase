<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="my-2 space-y-2 divide-y divide-block-border">
    <BBTableTabFilter
      v-if="isCurrentUserDBAOrOwner"
      class="mx-2"
      :tabList="['General', 'Membership']"
      :selectedIndex="state.selectedIndex"
      @select-index="
        (index) => {
          state.selectedIndex = index;
        }
      "
    />
    <div>
      <div class="px-4 py-2 flex justify-between">
        <BBSwitch
          :label="'Display all messages'"
          :value="state.showAll"
          @toggle="
            (on) => {
              showAll(on);
            }
          "
        />
        <button type="button" class="btn-normal" @click.prevent="markAllAsRead">
          <!-- Heroicon name: solid/pencil -->
          <svg
            class="-ml-1 mr-2 h-5 w-5 text-control-light"
            fill="currentColor"
            viewBox="0 0 20 20"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
            ></path>
          </svg>
          <span>Mark all as read</span>
        </button>
      </div>
      <ul class="divide-y divide-block-border">
        <li
          v-for="(message, index) in effectiveMessageList"
          :key="index"
          class="p-3 hover:bg-control-bg-hover cursor-default"
          @click.prevent="clickMessage(message)"
        >
          <div class="flex space-x-3">
            <BBAvatar :size="'small'" :username="message.creator.name" />
            <div class="flex-1 space-y-1">
              <div class="flex w-full items-center justify-between space-x-2">
                <h3
                  class="text-sm font-base text-control-light flex flex-row whitespace-nowrap"
                >
                  <router-link
                    :to="`/u/${message.creator.id}`"
                    class="font-medium text-main hover:underline"
                    >{{ message.creator.name }}</router-link
                  >
                  <span class="ml-1"> {{ actionSentence(message) }}</span>
                  <template
                    v-if="
                      message.type == 'bb.msg.member.create' ||
                      message.type == 'bb.msg.member.invite' ||
                      message.type == 'bb.msg.member.join' ||
                      message.type == 'bb.msg.member.revoke' ||
                      message.type == 'bb.msg.member.updaterole'
                    "
                  >
                  </template>
                  <template
                    v-else-if="
                      message.type == 'bb.msg.environment.create' ||
                      message.type == 'bb.msg.environment.update' ||
                      message.type == 'bb.msg.environment.archive' ||
                      message.type == 'bb.msg.environment.restore'
                    "
                  >
                    <router-link
                      :to="`/environment#${message.containerId}`"
                      class="normal-link ml-1"
                    >
                      {{ message.payload.environmentName }}
                    </router-link>
                  </template>
                  <template
                    v-else-if="message.type == 'bb.msg.environment.delete'"
                  >
                    <span class="font-medium text-main ml-1">
                      {{ message.payload.environmentName }}
                    </span>
                  </template>
                  <template
                    v-else-if="
                      message.type == 'bb.msg.instance.create' ||
                      message.type == 'bb.msg.instance.update' ||
                      message.type == 'bb.msg.instance.archive' ||
                      message.type == 'bb.msg.instance.restore'
                    "
                  >
                    <router-link
                      :to="`/instance/${message.containerId}`"
                      class="normal-link ml-1"
                    >
                      {{ message.payload.instanceName }}
                    </router-link>
                  </template>
                  <template v-else-if="message.type == 'bb.msg.issue.assign'">
                    <router-link
                      :to="`/issue/${message.containerId}`"
                      class="normal-link ml-1"
                    >
                      {{ message.payload.issueName }}
                    </router-link>
                  </template>
                  <template
                    v-else-if="message.type == 'bb.msg.issue.status.update'"
                  >
                    <router-link
                      :to="`/issue/${message.containerId}`"
                      class="normal-link ml-1"
                    >
                      {{ message.payload.issueName }}
                    </router-link>
                  </template>
                  <template v-else-if="message.type == 'bb.msg.issue.comment'">
                    <router-link
                      :to="`/issue/${message.containerId}#activity${message.payload.commentId}`"
                      class="normal-link ml-1"
                    >
                      {{ message.payload.issueName }}
                    </router-link>
                  </template>
                  <span
                    v-if="message.status == 'DELIVERED'"
                    class="ml-2 mt-1 h-3 w-3 rounded-full bg-accent"
                  ></span>
                </h3>
                <p class="text-sm text-control">
                  {{ humanizeTs(message.createdTs) }}
                </p>
              </div>
              <div v-if="message.description" class="text-sm text-control">
                {{ message.description }}
              </div>
            </div>
          </div>
        </li>
        <!-- More items... -->
      </ul>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, onMounted, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import {
  MemberMessagePayload,
  Message,
  Principal,
  PrincipalId,
  IssueAssignMessagePayload,
  IssueUpdateStatusMessagePayload,
  UNKNOWN_ID,
} from "../types";
import { isDBAOrOwner, roleName } from "../utils";

const GENERAL_TAB = 0;
const MEMBERSHIP_TAB = 1;

interface LocalState {
  selectedIndex: number;
  showAll: boolean;
  messageList: Message[];
  // To maintain a stable view when user clicks an item.
  // When user clicks an item, we will set the item as "CONSUMED".
  // Without this logic, if the user stays on the display unread item view,
  // that item will suddenly be removed from the list, which is not ideal for UX.
  whitelist: Message[];
}

export default {
  name: "Inbox",
  components: {},
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      selectedIndex: 0,
      showAll: false,
      messageList: [],
      whitelist: [],
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareMessageList = () => {
      store
        .dispatch("message/fetchMessageListByUser", currentUser.value.id)
        .then((list: Message[]) => {
          state.messageList = list;
        })
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareMessageList);

    onMounted(() => {
      state.whitelist = [];
    });

    const isCurrentUserDBAOrOwner = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const effectiveMessageList = computed(() => {
      return state.messageList.filter((message: Message) => {
        if (
          (state.selectedIndex == GENERAL_TAB &&
            message.type.startsWith("bb.msg.member.")) ||
          (state.selectedIndex == MEMBERSHIP_TAB &&
            !message.type.startsWith("bb.msg.member."))
        ) {
          return false;
        }

        return (
          state.showAll ||
          message.status == "DELIVERED" ||
          state.whitelist.find((item: Message) => {
            return item.id == message.id;
          })
        );
      });
    });

    const principalFromId = (principalId: PrincipalId): Principal => {
      return store.getters["principal/principalById"](principalId);
    };

    const actionSentence = (message: Message): string => {
      switch (message.type) {
        case "bb.msg.member.create": {
          const payload = message.payload as MemberMessagePayload;
          return `added ${
            principalFromId(payload.principalId).email
          } as ${roleName(payload.newRole!)}`;
        }
        case "bb.msg.member.invite": {
          const payload = message.payload as MemberMessagePayload;
          return `invited ${
            principalFromId(payload.principalId).email
          } as ${roleName(payload.newRole!)}`;
        }
        case "bb.msg.member.join": {
          const payload = message.payload as MemberMessagePayload;
          return `joined as ${roleName(payload.newRole!)}`;
        }
        case "bb.msg.member.revoke": {
          const payload = message.payload as MemberMessagePayload;
          return `revoked ${roleName(payload.oldRole!)} from ${
            principalFromId(payload.principalId).name
          }`;
        }
        case "bb.msg.member.updaterole":
          const payload = message.payload as MemberMessagePayload;
          return `changed ${
            principalFromId(payload.principalId).name
          } role from ${roleName(payload.oldRole!)} to ${roleName(
            payload.newRole!
          )}`;
        case "bb.msg.environment.create":
          return "created environment";
        case "bb.msg.environment.update":
          return "updated environment";
        case "bb.msg.environment.delete":
          return "deleted environment";
        case "bb.msg.environment.archive":
          return "archived environment";
        case "bb.msg.environment.restore":
          return "restored environment";
        case "bb.msg.environment.reorder":
          return "reordered environment";
        case "bb.msg.instance.create":
          return "created instance";
        case "bb.msg.instance.update":
          return "updated instance";
        case "bb.msg.instance.archive":
          return "archived instance";
        case "bb.msg.instance.restore":
          return "restored instance";
        case "bb.msg.issue.assign": {
          const payload = message.payload as IssueAssignMessagePayload;
          if (
            payload.oldAssigneeId == UNKNOWN_ID &&
            payload.newAssigneeId != UNKNOWN_ID
          ) {
            return "assigned " + currentUser.value.id == payload.newAssigneeId
              ? "you"
              : principalFromId(payload.newAssigneeId).name + " issue";
          } else if (
            payload.oldAssigneeId != UNKNOWN_ID &&
            payload.newAssigneeId == UNKNOWN_ID
          ) {
            return "un-assigned " + currentUser.value.id ==
              payload.oldAssigneeId
              ? "you"
              : principalFromId(payload.oldAssigneeId).name + " issue";
          } else if (
            payload.oldAssigneeId != UNKNOWN_ID &&
            payload.newAssigneeId != UNKNOWN_ID
          ) {
            return "re-assigned from " + currentUser.value.id ==
              payload.oldAssigneeId
              ? "you"
              : principalFromId(payload.oldAssigneeId).name +
                  " to " +
                  currentUser.value.id ==
                payload.newAssigneeId
              ? "you"
              : principalFromId(payload.newAssigneeId).name + " issue";
          }
          return "assigned issue";
        }
        case "bb.msg.issue.status.update": {
          const payload = message.payload as IssueUpdateStatusMessagePayload;
          return (
            "changed issue status from " +
            payload.oldStatus +
            " to " +
            payload.newStatus
          );
        }
        case "bb.msg.issue.stage.status.update": {
          const payload = message.payload as IssueUpdateStatusMessagePayload;
          return (
            "changed issue stage status from " +
            payload.oldStatus +
            " to " +
            payload.newStatus
          );
        }
        case "bb.msg.issue.comment":
          return "commented issue";
      }
    };

    const clickMessage = (message: Message) => {
      if (message.status == "DELIVERED") {
        state.whitelist.push(message);
        store.dispatch("message/patchMessage", {
          messageId: message.id,
          messagePatch: {
            updaterId: currentUser.value.id,
            status: "CONSUMED",
          },
        });
      }
    };

    const showAll = (show: boolean) => {
      state.whitelist = [];
      state.showAll = show;
    };

    const markAllAsRead = () => {
      state.messageList.forEach((item: Message) => {
        if (item.status == "DELIVERED") {
          state.whitelist.push(item);
          store.dispatch("message/patchMessage", {
            messageId: item.id,
            messagePatch: {
              updaterId: currentUser.value.id,
              status: "CONSUMED",
            },
          });
        }
      });
    };

    return {
      state,
      currentUser,
      principalFromId,
      isCurrentUserDBAOrOwner,
      effectiveMessageList,
      actionSentence,
      clickMessage,
      showAll,
      markAllAsRead,
    };
  },
};
</script>
