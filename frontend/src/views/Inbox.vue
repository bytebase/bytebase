<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="mx-4">
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
                <span class="font-medium text-main">{{
                  message.creator.name
                }}</span>
                <template v-if="message.type == 'bb.msg.task.assign'">
                  <span class="text-sm ml-1">
                    <template
                      v-if="
                        message.payload.oldAssigneeId == '-1' &&
                        message.payload.newAssigneeId != '-1'
                      "
                    >
                      assigned
                      {{
                        currentUser.id == message.payload.newAssigneeId
                          ? "you"
                          : principalFromId(message.payload.newAssigneeId).name
                      }}
                      task
                    </template>
                    <template
                      v-else-if="
                        message.payload.oldAssigneeId != '-1' &&
                        message.payload.newAssigneeId != '-1'
                      "
                    >
                      re-assigned from
                      {{
                        currentUser.id == message.payload.oldAssigneeId
                          ? "you"
                          : principalFromId(message.payload.oldAssigneeId).name
                      }}
                      to
                      {{
                        currentUser.id == message.payload.newAssigneeId
                          ? "you"
                          : principalFromId(message.payload.newAssigneeId).name
                      }}
                      task
                    </template>
                    <template
                      v-else-if="
                        message.payload.oldAssigneeId != '-1' &&
                        message.payload.newAssigneeId == '-1'
                      "
                    >
                      un-assigned
                      {{
                        currentUser.id == message.payload.oldAssigneeId
                          ? "you"
                          : principalFromId(message.payload.oldAssigneeId).name
                      }}
                      task
                    </template>
                  </span>
                  <router-link
                    :to="`/task/${message.containerId}`"
                    class="normal-link ml-1"
                  >
                    {{ message.payload.taskName }}
                  </router-link>
                </template>
                <template
                  v-else-if="message.type == 'bb.msg.task.updatestatus'"
                >
                  <span class="ml-1">
                    changed task status from {{ message.payload.oldStatus }} to
                    {{ message.payload.newStatus }}</span
                  >
                  <router-link
                    :to="`/task/${message.containerId}`"
                    class="normal-link ml-1"
                  >
                    {{ message.payload.taskName }}
                  </router-link>
                </template>
                <template v-else-if="message.type == 'bb.msg.task.comment'">
                  <span class="ml-1"> commented task</span>
                  <router-link
                    :to="`/task/${message.containerId}`"
                    class="normal-link ml-1"
                  >
                    {{ message.payload.taskName }}
                  </router-link>
                </template>
                <template v-else-if="message.type == 'bb.msg.instance.create'">
                  <span class="ml-1"> created instance</span>
                  <router-link
                    :to="`/instance/${message.containerId}`"
                    class="normal-link ml-1"
                  >
                    {{ message.payload.instanceName }}
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
</template>

<script lang="ts">
import { computed, onMounted, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import {
  Message,
  Principal,
  PrincipalId,
  TaskAssignMessagePayload,
  TaskUpdateStatusMessagePayload,
} from "../types";

interface LocalState {
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

    const effectiveMessageList = computed(() => {
      return state.showAll
        ? state.messageList
        : state.messageList.filter((message: Message) => {
            return (
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

    const clickMessage = (message: Message) => {
      if (message.status == "DELIVERED") {
        state.whitelist.push(message);
        store.dispatch("message/updateStatus", {
          messageId: message.id,
          updatedStatus: "CONSUMED",
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
          store.dispatch("message/updateStatus", {
            messageId: item.id,
            updatedStatus: "CONSUMED",
          });
        }
      });
    };

    return {
      state,
      currentUser,
      principalFromId,
      effectiveMessageList,
      clickMessage,
      showAll,
      markAllAsRead,
    };
  },
};
</script>
