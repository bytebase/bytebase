<template>
  <div class="space-y-4">
    <div v-if="create">
      <label class="textlabel"> Destination </label>
      <div class="mt-1 grid grid-cols-1 gap-4 sm:grid-cols-6">
        <template
          v-for="(item, index) in PROJECT_HOOK_TYPE_ITEM_LIST"
          :key="index"
        >
          <div
            class="
              flex
              justify-center
              px-2
              py-4
              border border-control-border
              hover:bg-control-bg-hover
              cursor-pointer
            "
            @click.capture="state.webhook.type = item.type"
          >
            <div class="flex flex-col items-center">
              <!-- This awkward code is author couldn't figure out proper way to use dynamic src under vite
                   https://github.com/vitejs/vite/issues/1265 -->
              <template v-if="item.type == 'bb.plugin.webhook.slack'">
                <img class="h-10 w-10" src="../assets/slack-logo.png" alt="" />
              </template>
              <template v-else-if="item.type == 'bb.plugin.webhook.discord'">
                <img class="h-10 w-10" src="../assets/discord-logo.svg" />
              </template>
              <template v-else-if="item.type == 'bb.plugin.webhook.teams'">
                <img class="h-10 w-10" src="../assets/teams-logo.svg" />
              </template>
              <template v-else-if="item.type == 'bb.plugin.webhook.dingtalk'">
                <img class="h-10 w-10" src="../assets/dingtalk-logo.png" />
              </template>
              <template v-else-if="item.type == 'bb.plugin.webhook.feishu'">
                <img class="h-10 w-10" src="../assets/feishu-logo.png" />
              </template>
              <template v-else-if="item.type == 'bb.plugin.webhook.wecom'">
                <img class="h-10 w-10" src="../assets/wecom-logo.png" />
              </template>
              <p class="mt-1 text-center textlabel">
                {{ item.name }}
              </p>
              <div class="mt-3 radio text-sm">
                <input
                  type="radio"
                  class="btn"
                  :checked="state.webhook.type == item.type"
                />
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
    <div>
      <label for="name" class="textlabel">
        Name <span class="text-red-600">*</span>
      </label>
      <input
        id="name"
        name="name"
        type="text"
        class="textfield mt-1 w-full"
        :placeholder="namePlaceholder"
        :disabled="!allowEdit"
        v-model="state.webhook.name"
      />
    </div>
    <div>
      <label for="url" class="textlabel">
        Webhook URL <span class="text-red-600">*</span>
      </label>
      <div class="mt-1 textinfolabel">
        <template v-if="state.webhook.type == 'bb.plugin.webhook.slack'">
          Create the corresponding webhook for the Slack channel receiving the
          message.
          <a
            href="https://api.slack.com/messaging/webhooks"
            target="__blank"
            class="normal-link"
            >View Slack's doc</a
          >.
        </template>
        <template v-else-if="state.webhook.type == 'bb.plugin.webhook.discord'">
          Create the corresponding webhook for the Discord channel receiving the
          message.
          <a
            href="https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks"
            target="__blank"
            class="normal-link"
            >View Discord's doc</a
          >.
        </template>
        <template v-else-if="state.webhook.type == 'bb.plugin.webhook.teams'">
          Create the corresponding webhook for the Teams channel receiving the
          message.
          <a
            href="https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook"
            target="__blank"
            class="normal-link"
            >View Teams's doc</a
          >.
        </template>
        <template
          v-else-if="state.webhook.type == 'bb.plugin.webhook.dingtalk'"
        >
          Create the corresponding webhook for the DingTalk group receiving the
          message. If you want to use keyword list to protect the webhook, you
          can add "Bytebase" to that list.
          <a
            href="https://developers.dingtalk.com/document/robots/custom-robot-access"
            target="__blank"
            class="normal-link"
            >View DingTalk's doc</a
          >.
        </template>
        <template v-else-if="state.webhook.type == 'bb.plugin.webhook.feishu'">
          Create the corresponding webhook for the Feishu group receiving the
          message. If you want to use keyword list to protect the webhook, you
          can add "Bytebase" to that list.
          <a
            href="https://www.feishu.cn/hc/zh-CN/articles/360024984973"
            target="__blank"
            class="normal-link"
            >View Feishu's doc</a
          >.
        </template>
        <!-- WeCom doesn't seem to provide official webhook setup guide for the enduser -->
        <template v-else-if="state.webhook.type == 'bb.plugin.webhook.wecom'">
          Create the corresponding webhook for the WeCom group receiving the
          message.
        </template>
      </div>
      <input
        id="url"
        name="url"
        type="text"
        class="textfield mt-1 w-full"
        :placeholder="urlPlaceholder"
        :disabled="!allowEdit"
        v-model="state.webhook.url"
      />
    </div>
    <div>
      <div class="text-md leading-6 font-medium text-main">
        Triggering activities
      </div>
      <div
        v-for="(item, index) in PROJECT_HOOK_ACTIVITY_ITEM_LIST"
        :key="index"
        class="mt-4 space-y-4"
      >
        <BBCheckbox
          :title="item.title"
          :label="item.label"
          :value="eventOn(item.activity)"
          @toggle="
            (on) => {
              toggleEvent(item.activity, on);
            }
          "
        />
      </div>
    </div>
    <div
      class="flex pt-5"
      :class="!create && allowEdit ? 'justify-between' : 'justify-end'"
    >
      <BBButtonConfirm
        v-if="!create && allowEdit"
        :style="'DELETE'"
        :buttonText="'Delete this webhook'"
        :okText="'Delete'"
        :confirmTitle="`Delete webhook '${webhook.name}' and all its execution history?`"
        :requireConfirm="true"
        @confirm="deleteWebhook"
      />
      <div>
        <button type="button" class="btn-normal" @click.prevent="cancel">
          {{ allowEdit ? "Cancel" : "Back" }}
        </button>
        <template v-if="allowEdit">
          <button
            v-if="create"
            type="submit"
            class="btn-primary ml-3"
            :disabled="!allowCreate"
            @click.prevent="createWebhook"
          >
            Create
          </button>
          <button
            v-else
            type="submit"
            class="btn-primary ml-3"
            :disabled="!valueChanged"
            @click.prevent="updateWebhook"
          >
            Update
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "@vue/reactivity";
import { computed, PropType, watch } from "@vue/runtime-core";
import {
  ActivityType,
  Project,
  ProjectWebhook,
  ProjectWebhookCreate,
  ProjectWebhookPatch,
  PROJECT_HOOK_TYPE_ITEM_LIST,
  PROJECT_HOOK_ACTIVITY_ITEM_LIST,
} from "../types";
import { cloneDeep, isEmpty, isEqual } from "lodash";
import { useRouter } from "vue-router";
import { projectWebhookSlug, projectSlug } from "../utils";
import { useStore } from "vuex";

interface LocalState {
  webhook: ProjectWebhook | ProjectWebhookCreate;
}

export default {
  name: "ProjectWebhookForm",
  emits: ["change-repository"],
  props: {
    allowEdit: {
      default: true,
      type: Boolean,
    },
    create: {
      type: Boolean,
      default: false,
    },
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
    webhook: {
      required: true,
      type: Object as PropType<ProjectWebhook | ProjectWebhookCreate>,
    },
  },
  components: {},
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      webhook: cloneDeep(props.webhook),
    });

    watch(
      () => props.webhook,
      (cur: ProjectWebhook | ProjectWebhookCreate) => {
        state.webhook = cloneDeep(cur);
      }
    );

    const namePlaceholder = computed(() => {
      if (state.webhook.type == "bb.plugin.webhook.slack") {
        return "Slack Webhook";
      } else if (state.webhook.type == "bb.plugin.webhook.discord") {
        return "Discord Webhook";
      } else if (state.webhook.type == "bb.plugin.webhook.teams") {
        return "Teams Webhook";
      } else if (state.webhook.type == "bb.plugin.webhook.dingtalk") {
        return "DingTalk Webhook";
      } else if (state.webhook.type == "bb.plugin.webhook.feishu") {
        return "Feishu Webhook";
      } else if (state.webhook.type == "bb.plugin.webhook.wecom") {
        return "WeCom Webhook";
      }

      return "My Webhook";
    });

    const urlPlaceholder = computed(() => {
      if (state.webhook.type == "bb.plugin.webhook.slack") {
        return "https://hooks.slack.com/services/...";
      } else if (state.webhook.type == "bb.plugin.webhook.discord") {
        return "https://discord.com/api/webhooks/...";
      } else if (state.webhook.type == "bb.plugin.webhook.teams") {
        return "https://acme123.webhook.office.com/webhookb2/...";
      } else if (state.webhook.type == "bb.plugin.webhook.dingtalk") {
        return "https://oapi.dingtalk.com/robot/...";
      } else if (state.webhook.type == "bb.plugin.webhook.feishu") {
        return "https://open.feishu.cn/open-apis/bot/v2/hook/...";
      } else if (state.webhook.type == "bb.plugin.webhook.wecom") {
        return "https://qyapi.weixin.qq.com/cgi-bin/webhook/...";
      }

      return "Webhook URL";
    });

    const valueChanged = computed(() => {
      return !isEqual(props.webhook, state.webhook);
    });

    const allowCreate = computed(() => {
      return (
        !isEmpty(state.webhook.type) &&
        !isEmpty(state.webhook.name) &&
        !isEmpty(state.webhook.url)
      );
    });

    const eventOn = (type: ActivityType) => {
      for (const activityType of props.webhook.activityList) {
        if (activityType == type) {
          return true;
        }
      }
      return false;
    };

    const toggleEvent = (type: ActivityType, on: boolean) => {
      if (on) {
        for (const activityType of state.webhook.activityList) {
          if (activityType == type) {
            return;
          }
        }
        state.webhook.activityList.push(type);
      } else {
        const list: ActivityType[] = [];
        for (const activityType of state.webhook.activityList) {
          if (activityType != type) {
            list.push(activityType);
          }
        }
        state.webhook.activityList = list;
      }
      state.webhook.activityList.sort();
    };

    const cancel = () => {
      router.push({
        name: "workspace.project.detail",
        params: {
          projectSlug: projectSlug(props.project),
        },
        hash: "#webhook",
      });
    };

    const createWebhook = () => {
      store
        .dispatch("projectWebhook/createProjectWebhook", {
          projectID: props.project.id,
          projectWebhookCreate: state.webhook,
        })
        .then((webhook: ProjectWebhook) => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully created webhook "${webhook.name}".`,
          });
          router.push({
            name: "workspace.project.hook.detail",
            params: {
              projectWebhookSlug: projectWebhookSlug(webhook),
            },
          });
        });
    };

    const updateWebhook = () => {
      const projectWebhookPatch: ProjectWebhookPatch = {};
      if (props.webhook.name != state.webhook.name) {
        projectWebhookPatch.name = state.webhook.name;
      }
      if (props.webhook.url != state.webhook.url) {
        projectWebhookPatch.url = state.webhook.url;
      }
      if (props.webhook.activityList != state.webhook.activityList) {
        projectWebhookPatch.activityList = state.webhook.activityList.join(",");
      }
      store
        .dispatch("projectWebhook/updateProjectWebhookByID", {
          projectID: props.project.id,
          projectWebhookID: (state.webhook as ProjectWebhook).id,
          projectWebhookPatch,
        })
        .then((webhook: ProjectWebhook) => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully updated webhook "${webhook.name}".`,
          });
        });
    };

    const deleteWebhook = () => {
      const name = state.webhook.name;
      store
        .dispatch("projectWebhook/deleteProjectWebhookByID", {
          projectID: props.project.id,
          projectWebhookID: (state.webhook as ProjectWebhook).id,
        })
        .then(() => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully deleted webhook "${name}".`,
          });
          cancel();
        });
    };

    return {
      PROJECT_HOOK_TYPE_ITEM_LIST,
      PROJECT_HOOK_ACTIVITY_ITEM_LIST,
      state,
      namePlaceholder,
      urlPlaceholder,
      valueChanged,
      allowCreate,
      eventOn,
      toggleEvent,
      cancel,
      createWebhook,
      updateWebhook,
      deleteWebhook,
    };
  },
};
</script>
