<template>
  <div class="space-y-4">
    <div v-if="create">
      <label class="textlabel">
        {{ $t("project.webhook.destination") }}
      </label>
      <div class="mt-1 grid grid-cols-1 gap-4 sm:grid-cols-7">
        <template
          v-for="(item, index) in PROJECT_HOOK_TYPE_ITEM_LIST()"
          :key="index"
        >
          <div
            class="flex justify-center px-2 py-4 border border-control-border hover:bg-control-bg-hover cursor-pointer"
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
              <template v-else-if="item.type == 'bb.plugin.webhook.custom'">
                <img class="h-10 w-10" src="../assets/custom-logo.svg" />
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
        {{ $t("common.name") }} <span class="text-red-600">*</span>
      </label>
      <input
        id="name"
        v-model="state.webhook.name"
        name="name"
        type="text"
        class="textfield mt-1 w-full"
        :placeholder="namePlaceholder"
        :disabled="!allowEdit"
      />
    </div>
    <div>
      <label for="url" class="textlabel">
        {{ $t("project.webhook.webhook-url") }}
        <span class="text-red-600">*</span>
      </label>
      <div class="mt-1 textinfolabel">
        <template v-if="state.webhook.type == 'bb.plugin.webhook.slack'">
          {{
            $t("project.webhook.creation.desc", {
              destination: $t("common.slack"),
            })
          }}
          <a
            href="https://api.slack.com/messaging/webhooks"
            target="__blank"
            class="normal-link"
            >{{
              $t("project.webhook.creation.view-doc", {
                destination: $t("common.slack"),
              })
            }}</a
          >.
        </template>
        <template v-else-if="state.webhook.type == 'bb.plugin.webhook.discord'">
          {{
            $t("project.webhook.creation.desc", {
              destination: $t("common.discord"),
            })
          }}
          <a
            href="https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks"
            target="__blank"
            class="normal-link"
            >{{
              $t("project.webhook.creation.view-doc", {
                destination: $t("common.discord"),
              })
            }}</a
          >.
        </template>
        <template v-else-if="state.webhook.type == 'bb.plugin.webhook.teams'">
          {{
            $t("project.webhook.creation.desc", {
              destination: $t("common.teams"),
            })
          }}
          <a
            href="https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook"
            target="__blank"
            class="normal-link"
            >{{
              $t("project.webhook.creation.view-doc", {
                destination: $t("common.teams"),
              })
            }}</a
          >.
        </template>
        <template
          v-else-if="state.webhook.type == 'bb.plugin.webhook.dingtalk'"
        >
          {{
            $t("project.webhook.creation.desc", {
              destination: $t("common.dingtalk"),
            }) +
            ". " +
            $t("project.webhook.creation.how-to-protect")
          }}
          <a
            href="https://developers.dingtalk.com/document/robots/custom-robot-access"
            target="__blank"
            class="normal-link"
            >{{
              $t("project.webhook.creation.view-doc", {
                destination: $t("common.dingtalk"),
              })
            }}</a
          >.
        </template>
        <template v-else-if="state.webhook.type == 'bb.plugin.webhook.feishu'">
          {{
            $t("project.webhook.creation.desc", {
              destination: $t("common.feishu"),
            }) +
            ". " +
            $t("project.webhook.creation.how-to-protect")
          }}

          <a
            href="https://www.feishu.cn/hc/zh-CN/articles/360024984973"
            target="__blank"
            class="normal-link"
            >{{
              $t("project.webhook.creation.view-doc", {
                destination: $t("common.feishu"),
              })
            }}</a
          >.
        </template>
        <!-- WeCom doesn't seem to provide official webhook setup guide for the enduser -->
        <template v-else-if="state.webhook.type == 'bb.plugin.webhook.wecom'">
          {{
            $t("project.webhook.creation.desc", {
              destination: $t("common.wecom"),
            })
          }}
        </template>
        <template v-else-if="state.webhook.type == 'bb.plugin.webhook.custom'">
          {{
            $t("project.webhook.creation.desc", {
              destination: $t("common.custom"),
            })
          }}
          <a
            href="https://www.bytebase.com/docs/use-bytebase/webhook-integration/project-webhook#custom"
            target="__blank"
            class="normal-link"
            >{{
              $t("project.webhook.creation.view-doc", {
                destination: $t("common.custom"),
              })
            }}</a
          >.
        </template>
      </div>
      <input
        id="url"
        v-model="state.webhook.url"
        name="url"
        type="text"
        class="textfield mt-1 w-full"
        :placeholder="urlPlaceholder"
        :disabled="!allowEdit"
      />
    </div>
    <div>
      <div class="text-md leading-6 font-medium text-main">
        {{ $t("project.webhook.triggering-activity") }}
      </div>
      <div
        v-for="(item, index) in PROJECT_HOOK_ACTIVITY_ITEM_LIST()"
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
        :button-text="$t('project.webhook.deletion.btn-text')"
        :ok-text="'Delete'"
        :confirm-title="`Delete webhook '${webhook.name}' and all its execution history?`"
        :require-confirm="true"
        @confirm="deleteWebhook"
      />
      <div>
        <button type="button" class="btn-normal" @click.prevent="cancel">
          {{ allowEdit ? $t("common.cancel") : $t("common.back") }}
        </button>
        <template v-if="allowEdit">
          <button
            v-if="create"
            type="submit"
            class="btn-primary ml-3"
            :disabled="!allowCreate"
            @click.prevent="createWebhook"
          >
            {{ $t("common.create") }}
          </button>
          <button
            v-else
            type="submit"
            class="btn-primary ml-3"
            :disabled="!valueChanged"
            @click.prevent="updateWebhook"
          >
            {{ $t("common.update") }}
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, computed, PropType, watch, defineComponent } from "vue";
import {
  ActivityType,
  Project,
  ProjectWebhook,
  ProjectWebhookCreate,
  ProjectWebhookPatch,
  PROJECT_HOOK_TYPE_ITEM_LIST,
  PROJECT_HOOK_ACTIVITY_ITEM_LIST,
} from "../types";
import { cloneDeep, isEmpty, isEqual } from "lodash-es";
import { useRouter } from "vue-router";
import { projectWebhookSlug, projectSlug } from "../utils";
import { useI18n } from "vue-i18n";
import { pushNotification, useProjectWebhookStore } from "@/store";

interface LocalState {
  webhook: ProjectWebhook | ProjectWebhookCreate;
}

export default defineComponent({
  name: "ProjectWebhookForm",
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
  emits: ["change-repository"],
  setup(props) {
    const router = useRouter();
    const { t } = useI18n();
    const projectWebhookStore = useProjectWebhookStore();

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
        return `${t("common.slack")} Webhook`;
      } else if (state.webhook.type == "bb.plugin.webhook.discord") {
        return `${t("common.discord")} Webhook`;
      } else if (state.webhook.type == "bb.plugin.webhook.teams") {
        return `${t("common.teams")} Webhook`;
      } else if (state.webhook.type == "bb.plugin.webhook.dingtalk") {
        return `${t("common.dingtalk")} Webhook`;
      } else if (state.webhook.type == "bb.plugin.webhook.feishu") {
        return `${t("common.feishu")} Webhook`;
      } else if (state.webhook.type == "bb.plugin.webhook.wecom") {
        return `${t("common.wecom")} Webhook`;
      } else if (state.webhook.type == "bb.plugin.webhook.custom") {
        return `${t("common.custom")} Webhook`;
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
      } else if (state.webhook.type == "bb.plugin.webhook.custom") {
        return "https://example.com/...";
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
      projectWebhookStore
        .createProjectWebhook({
          projectId: props.project.id,
          projectWebhookCreate: state.webhook,
        })
        .then((webhook: ProjectWebhook) => {
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("project.webhook.success-created-prompt", {
              name: webhook.name,
            }),
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
      projectWebhookStore
        .updateProjectWebhookById({
          projectId: props.project.id,
          projectWebhookId: (state.webhook as ProjectWebhook).id,
          projectWebhookPatch,
        })
        .then((webhook: ProjectWebhook) => {
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("project.webhook.success-updated-prompt", {
              name: webhook.name,
            }),
          });
        });
    };

    const deleteWebhook = () => {
      const name = state.webhook.name;
      projectWebhookStore
        .deleteProjectWebhookById({
          projectId: props.project.id,
          projectWebhookId: (state.webhook as ProjectWebhook).id,
        })
        .then(() => {
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("project.webhook.success-deleted-prompt", {
              name: name,
            }),
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
});
</script>
