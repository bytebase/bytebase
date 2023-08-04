<template>
  <div class="space-y-4">
    <div v-if="create">
      <label class="textlabel">
        {{ $t("project.webhook.destination") }}
      </label>
      <div class="mt-1 grid grid-cols-1 gap-4 sm:grid-cols-7">
        <template
          v-for="(item, index) in projectWebhookV1TypeItemList()"
          :key="index"
        >
          <div
            class="flex justify-center px-2 py-4 border border-control-border hover:bg-control-bg-hover cursor-pointer"
            @click.capture="state.webhook.type = item.type"
          >
            <div class="flex flex-col items-center">
              <!-- This awkward code is author couldn't figure out proper way to use dynamic src under vite
                   https://github.com/vitejs/vite/issues/1265 -->
              <template v-if="item.type === Webhook_Type.TYPE_SLACK">
                <img class="h-10 w-10" src="../assets/slack-logo.png" alt="" />
              </template>
              <template v-else-if="item.type === Webhook_Type.TYPE_DISCORD">
                <img class="h-10 w-10" src="../assets/discord-logo.svg" />
              </template>
              <template v-else-if="item.type === Webhook_Type.TYPE_TEAMS">
                <img class="h-10 w-10" src="../assets/teams-logo.svg" />
              </template>
              <template v-else-if="item.type === Webhook_Type.TYPE_DINGTALK">
                <img class="h-10 w-10" src="../assets/dingtalk-logo.png" />
              </template>
              <template v-else-if="item.type === Webhook_Type.TYPE_FEISHU">
                <img class="h-10 w-10" src="../assets/feishu-logo.webp" />
              </template>
              <template v-else-if="item.type === Webhook_Type.TYPE_WECOM">
                <img class="h-10 w-10" src="../assets/wecom-logo.png" />
              </template>
              <template v-else-if="item.type === Webhook_Type.TYPE_CUSTOM">
                <heroicons-outline:puzzle class="w-10 h-10" />
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
        v-model="state.webhook.title"
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
        <template v-if="state.webhook.type === Webhook_Type.TYPE_SLACK">
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
        <template v-else-if="state.webhook.type === Webhook_Type.TYPE_DISCORD">
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
        <template v-else-if="state.webhook.type === Webhook_Type.TYPE_TEAMS">
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
        <template v-else-if="state.webhook.type === Webhook_Type.TYPE_DINGTALK">
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
        <template v-else-if="state.webhook.type === Webhook_Type.TYPE_FEISHU">
          {{
            $t("project.webhook.creation.desc", {
              destination: $t("common.feishu"),
            }) +
            ". " +
            $t("project.webhook.creation.how-to-protect")
          }}

          <a
            href="https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot"
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
        <template v-else-if="state.webhook.type === Webhook_Type.TYPE_WECOM">
          {{
            $t("project.webhook.creation.desc", {
              destination: $t("common.wecom"),
            })
          }}
          <a
            href="https://open.work.weixin.qq.com/help2/pc/14931"
            target="__blank"
            class="normal-link"
            >{{
              $t("project.webhook.creation.view-doc", {
                destination: $t("common.wecom"),
              })
            }}</a
          >.
        </template>
        <template v-else-if="state.webhook.type === Webhook_Type.TYPE_CUSTOM">
          {{
            $t("project.webhook.creation.desc", {
              destination: $t("common.custom"),
            })
          }}
          <a
            href="https://www.bytebase.com/docs/change-database/webhook#custom?source=console"
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
        v-for="(item, index) in projectWebhookV1ActivityItemList()"
        :key="index"
        class="mt-4 space-y-4"
      >
        <BBCheckbox
          :title="item.title"
          :label="item.label"
          :value="isEventOn(item.activity)"
          @toggle="
            (on: boolean) => {
              toggleEvent(item.activity, on);
            }
          "
        />
      </div>
    </div>
    <button
      type="button"
      class="btn-normal whitespace-nowrap items-center"
      @click.prevent="testWebhook"
    >
      {{ $t("project.webhook.test-webhook") }}
    </button>
    <div
      class="flex pt-5"
      :class="!create && allowEdit ? 'justify-between' : 'justify-end'"
    >
      <BBButtonConfirm
        v-if="!create && allowEdit"
        :style="'DELETE'"
        :button-text="$t('project.webhook.deletion.btn-text')"
        :ok-text="$t('common.delete')"
        :confirm-title="
          $t('project.webhook.deletion.confirm-title', { title: webhook.title })
        "
        :confirm-description="$t('common.cannot-undo-this-action')"
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
            :disabled="
              !valueChanged || state.webhook.notificationTypes.length === 0
            "
            @click.prevent="updateWebhook"
          >
            {{ $t("common.update") }}
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed, PropType, watch } from "vue";
import { cloneDeep, isEmpty, isEqual } from "lodash-es";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";

import { projectV1Slug, projectWebhookV1Slug } from "../utils";
import {
  pushNotification,
  useProjectWebhookV1Store,
  useGracefulRequest,
} from "@/store";
import {
  Activity_Type,
  Project,
  Webhook,
  Webhook_Type,
} from "@/types/proto/v1/project_service";
import {
  projectWebhookV1ActivityItemList,
  projectWebhookV1TypeItemList,
} from "@/types";

interface LocalState {
  webhook: Webhook;
}

const props = defineProps({
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
    type: Object as PropType<Webhook>,
  },
});

const router = useRouter();
const { t } = useI18n();

const projectWebhookV1Store = useProjectWebhookV1Store();
const state = reactive<LocalState>({
  webhook: cloneDeep(props.webhook),
});

watch(
  () => props.webhook,
  (webhook) => {
    state.webhook = cloneDeep(webhook);
  }
);

const namePlaceholder = computed(() => {
  if (state.webhook.type === Webhook_Type.TYPE_SLACK) {
    return `${t("common.slack")} Webhook`;
  } else if (state.webhook.type === Webhook_Type.TYPE_DISCORD) {
    return `${t("common.discord")} Webhook`;
  } else if (state.webhook.type === Webhook_Type.TYPE_TEAMS) {
    return `${t("common.teams")} Webhook`;
  } else if (state.webhook.type === Webhook_Type.TYPE_DINGTALK) {
    return `${t("common.dingtalk")} Webhook`;
  } else if (state.webhook.type === Webhook_Type.TYPE_FEISHU) {
    return `${t("common.feishu")} Webhook`;
  } else if (state.webhook.type === Webhook_Type.TYPE_WECOM) {
    return `${t("common.wecom")} Webhook`;
  } else if (state.webhook.type === Webhook_Type.TYPE_CUSTOM) {
    return `${t("common.custom")} Webhook`;
  }

  return "My Webhook";
});

const urlPlaceholder = computed(() => {
  if (state.webhook.type === Webhook_Type.TYPE_SLACK) {
    return "https://hooks.slack.com/services/...";
  } else if (state.webhook.type === Webhook_Type.TYPE_DISCORD) {
    return "https://discord.com/api/webhooks/...";
  } else if (state.webhook.type === Webhook_Type.TYPE_TEAMS) {
    return "https://acme123.webhook.office.com/webhookb2/...";
  } else if (state.webhook.type === Webhook_Type.TYPE_DINGTALK) {
    return "https://oapi.dingtalk.com/robot/...";
  } else if (state.webhook.type === Webhook_Type.TYPE_FEISHU) {
    return "https://open.feishu.cn/open-apis/bot/v2/hook/...";
  } else if (state.webhook.type === Webhook_Type.TYPE_WECOM) {
    return "https://qyapi.weixin.qq.com/cgi-bin/webhook/...";
  } else if (state.webhook.type === Webhook_Type.TYPE_CUSTOM) {
    return "https://example.com/api/webhook/...";
  }

  return "Webhook URL";
});

const valueChanged = computed(() => {
  return !isEqual(props.webhook, state.webhook);
});

const allowCreate = computed(() => {
  return (
    !isEmpty(state.webhook.title) &&
    !isEmpty(state.webhook.url) &&
    !isEmpty(state.webhook.notificationTypes)
  );
});

const isEventOn = (type: Activity_Type) => {
  return props.webhook.notificationTypes.includes(type);
};

const toggleEvent = (type: Activity_Type, on: boolean) => {
  if (on) {
    if (state.webhook.notificationTypes.includes(type)) {
      return;
    }
    state.webhook.notificationTypes.push(type);
  } else {
    const index = state.webhook.notificationTypes.indexOf(type);
    if (index >= 0) {
      state.webhook.notificationTypes.splice(index, 1);
    }
  }
  state.webhook.notificationTypes.sort();
};

const cancel = () => {
  router.push({
    name: "workspace.project.detail",
    params: {
      projectSlug: projectV1Slug(props.project),
    },
    hash: "#webhook",
  });
};

const createWebhook = () => {
  useGracefulRequest(async () => {
    const { webhook } = state;
    const updatedProject = await projectWebhookV1Store.createProjectWebhook(
      props.project,
      webhook
    );

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.webhook.success-created-prompt", {
        name: webhook.title,
      }),
    });
    const createdWebhook = updatedProject.webhooks.find((wh) => {
      return (
        wh.title === webhook.title &&
        wh.type == webhook.type &&
        wh.url === webhook.url
      );
    });
    if (createdWebhook) {
      router.push({
        name: "workspace.project.hook.detail",
        params: {
          projectWebhookSlug: projectWebhookV1Slug(createdWebhook),
        },
      });
    }
  });
};

const updateWebhook = () => {
  useGracefulRequest(async () => {
    const updateMask: string[] = [];
    if (state.webhook.title !== props.webhook.title) {
      updateMask.push("title");
    }
    if (state.webhook.url !== props.webhook.url) {
      updateMask.push("url");
    }
    if (
      !isEqual(state.webhook.notificationTypes, props.webhook.notificationTypes)
    ) {
      updateMask.push("notification_type");
    }
    await projectWebhookV1Store.updateProjectWebhook(state.webhook, updateMask);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.webhook.success-updated-prompt", {
        name: state.webhook.title,
      }),
    });
  });
};

const deleteWebhook = () => {
  useGracefulRequest(async () => {
    const name = state.webhook.title;
    await projectWebhookV1Store.deleteProjectWebhook(state.webhook);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.webhook.success-deleted-prompt", {
        name,
      }),
    });
    cancel();
  });
};

const testWebhook = () => {
  useGracefulRequest(async () => {
    console.log("Barny1", props.project);
    console.log("Barny2", state.webhook);
    const result = await useProjectWebhookV1Store().testProjectWebhook(
      props.project,
      state.webhook
    );

    if (result.error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("project.webhook.fail-tested-title"),
        description: result.error,
        manualHide: true,
      });
    } else {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.webhook.success-tested-prompt"),
      });
    }
  });
};
</script>
