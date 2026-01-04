<template>
  <FormLayout :title="create ? $t('project.webhook.creation.title') : ''">
    <template v-if="!create" #title>
      <div class="flex flex-row gap-x-2 items-center">
        <WebhookTypeIcon :type="props.webhook.type" class="h-6 w-6" />
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ props.webhook.title }}
        </h3>
      </div>
      <NDivider />
    </template>
    <template #body>
      <MissingExternalURLAttention
        class="mb-6"
      />
      <div class="flex flex-col gap-y-4">
        <div v-if="create">
          <div>
            <label for="name" class="font-medium text-main">
              {{ $t("project.webhook.destination") }}
              <RequiredStar />
            </label>
          </div>
          <NRadioGroup class="w-full mt-1" :value="state.webhook.type">
            <div class="grid grid-cols-1 gap-4 sm:grid-cols-7">
              <template
                v-for="(item, index) in webhookTypeItemList"
                :key="index"
              >
                <div
                  class="flex justify-center px-2 py-4 rounded-sm border border-control-border hover:bg-control-bg-hover cursor-pointer"
                  @click.capture="state.webhook.type = item.type"
                >
                  <div class="flex flex-col items-center">
                    <WebhookTypeIcon :type="item.type" class="h-10 w-10" />
                    <p class="mt-1 text-center textlabel">
                      {{ item.name }}
                    </p>
                    <div class="mt-3 radio text-sm">
                      <NRadio :value="item.type" />
                    </div>
                  </div>
                </div>
              </template>
            </div>
          </NRadioGroup>
        </div>
        <div>
          <label for="name" class="font-medium text-main">
            {{ $t("common.name") }}
            <RequiredStar />
          </label>
          <NInput
            id="name"
            v-model:value="state.webhook.title"
            name="name"
            class="mt-1 w-full"
            :placeholder="`${selectedWebhook?.name ?? 'My'} Webhook`"
            :disabled="!allowEdit"
          />
        </div>
        <div>
          <label for="url" class="font-medium text-main">
            {{ $t("project.webhook.webhook-url") }}
            <RequiredStar />
          </label>
          <div class="mt-1 textinfolabel">
            {{
              $t("project.webhook.creation.desc", {
                destination: selectedWebhook?.name,
              })
            }}
            <a
              :href="selectedWebhook?.docUrl"
              target="__blank"
              class="normal-link"
              >{{
                $t("project.webhook.creation.view-doc", {
                  destination: selectedWebhook?.name,
                })
              }}</a
            >.
          </div>
          <NInput
            id="url"
            v-model:value="state.webhook.url"
            name="url"
            class="mt-1 w-full"
            :placeholder="selectedWebhook?.urlPlaceholder"
            :disabled="!allowEdit"
          />
        </div>
        <div>
          <div class="text-md leading-6 font-medium text-main">
            {{ $t("project.webhook.triggering-activity") }}
            <RequiredStar />
          </div>
          <div class="flex flex-col gap-y-4 mt-2">
            <div v-for="(item, index) in webhookActivityItemList" :key="index">
              <div>
                <div class="flex items-center">
                  <NCheckbox
                    :label="item.title"
                    :checked="isEventOn(item.activity)"
                    @update:checked="
                      (on: boolean) => {
                        toggleEvent(item.activity, on);
                      }
                    "
                  />
                  <NTooltip
                    v-if="
                      webhookSupportDirectMessage && item.supportDirectMessage
                    "
                  >
                    <template #trigger>
                      <InfoIcon class="w-4 h-auto text-gray-500" />
                    </template>
                    {{ $t("project.webhook.activity-support-direct-message") }}
                  </NTooltip>
                </div>
                <div class="textinfolabel">{{ item.label }}</div>
              </div>
            </div>
          </div>
        </div>
        <div v-if="webhookSupportDirectMessage">
          <div class="text-md leading-6 font-medium text-main">
            {{ $t("project.webhook.direct-messages") }}
          </div>
          <BBAttention class="my-2" :type="imApp ? 'info' : 'warning'">
            <template #default>
              <span v-if="imApp">
                {{ $t("project.webhook.direct-messages-tip") }}
              </span>
              <i18n-t
                v-else
                class="textinfolabel"
                tag="div"
                keypath="project.webhook.direct-messages-warning"
              >
                <template #im>
                  <router-link
                    target="_blank"
                    class="normal-link"
                    :to="{ name: WORKSPACE_ROUTE_IM }"
                  >
                    {{ $t("settings.sidebar.im-integration") }}
                  </router-link>
                </template>
              </i18n-t>
            </template>
          </BBAttention>
          <span class="mt-1 textinfolabel">
            <i18n-t
              keypath="project.webhook.direct-messages-description"
              tag="span"
            >
              <template #events>
                <ul class="list-disc pl-4">
                  <li
                    v-for="(item, index) in webhookActivityItemList.filter(
                      (item) => item.supportDirectMessage
                    )"
                    :key="index"
                  >
                    {{ item.title }}
                  </li>
                </ul>
              </template>
            </i18n-t>
          </span>
          <div class="flex items-center mt-2">
            <NCheckbox
              v-model:checked="state.webhook.directMessage"
              :disabled="!activitySupportDirectMessage"
              :label="$t('project.webhook.enable-direct-messages')"
            />
          </div>
        </div>
        <div class="mt-4">
          <NButton @click.prevent="testWebhook">
            {{ $t("project.webhook.test-webhook") }}
          </NButton>
        </div>
      </div>
    </template>
    <template #footer>
      <div
        class="flex"
        :class="!create && allowEdit ? 'justify-between' : 'justify-end'"
      >
        <BBButtonConfirm
          v-if="!create && allowEdit"
          :type="'DELETE'"
          :button-text="$t('project.webhook.deletion.btn-text')"
          :ok-text="$t('common.delete')"
          :confirm-title="
            $t('project.webhook.deletion.confirm-title', {
              title: webhook.title,
            })
          "
          :confirm-description="$t('common.cannot-undo-this-action')"
          :require-confirm="true"
          @confirm="deleteWebhook"
        />
        <div class="flex items-center gap-x-3">
          <NButton v-if="create" @click.prevent="cancel">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton v-else-if="valueChanged" @click.prevent="discardChanges">
            {{ $t("common.discard-changes") }}
          </NButton>
          <template v-if="allowEdit">
            <NButton
              v-if="create"
              type="primary"
              :disabled="!allowCreate"
              @click.prevent="createWebhook"
            >
              {{ $t("common.create") }}
            </NButton>
            <NButton
              v-else
              type="primary"
              :disabled="
                !valueChanged || state.webhook.notificationTypes.length === 0
              "
              @click.prevent="updateWebhook"
            >
              {{ $t("common.update") }}
            </NButton>
          </template>
        </div>
      </div>
    </template>
  </FormLayout>
</template>

<script lang="ts" setup>
import { cloneDeep, isEmpty, isEqual } from "lodash-es";
import { InfoIcon } from "lucide-vue-next";
import {
  NButton,
  NCheckbox,
  NDivider,
  NInput,
  NRadio,
  NRadioGroup,
  NTooltip,
} from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAttention, BBButtonConfirm } from "@/bbkit";
import RequiredStar from "@/components/RequiredStar.vue";
import { MissingExternalURLAttention } from "@/components/v2/Form";
import FormLayout from "@/components/v2/Form/FormLayout.vue";
import { useBodyLayoutContext } from "@/layouts/common";
import {
  PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
  PROJECT_V1_ROUTE_WEBHOOKS,
} from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_IM } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useGracefulRequest,
  useProjectV1Store,
  useProjectWebhookV1Store,
  useSettingV1Store,
} from "@/store";
import {
  projectWebhookV1ActivityItemList,
  projectWebhookV1TypeItemList,
} from "@/types";
import {
  type Activity_Type,
  type Project,
  type Webhook,
} from "@/types/proto-es/v1/project_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { projectWebhookV1Slug } from "../utils";
import WebhookTypeIcon from "./Project/WebhookTypeIcon.vue";

interface LocalState {
  webhook: Webhook;
}

const props = withDefaults(
  defineProps<{
    allowEdit?: boolean;
    create: boolean;
    project: Project;
    webhook: Webhook;
  }>(),
  {
    allowEdit: true,
  }
);

const router = useRouter();
const { t } = useI18n();

const settingStore = useSettingV1Store();
const projectStore = useProjectV1Store();
const projectWebhookV1Store = useProjectWebhookV1Store();
const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("!pb-0");

const state = reactive<LocalState>({
  webhook: cloneDeep(props.webhook),
});

watch(
  () => props.webhook,
  (webhook) => {
    state.webhook = cloneDeep(webhook);
  }
);

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

const discardChanges = () => {
  state.webhook = cloneDeep(props.webhook);
};

const isEventOn = (type: Activity_Type) => {
  return state.webhook.notificationTypes.includes(type);
};

const webhookActivityItemList = computed(() =>
  projectWebhookV1ActivityItemList()
);

const webhookTypeItemList = computed(() => projectWebhookV1TypeItemList());

const selectedWebhook = computed(() => {
  return webhookTypeItemList.value.find(
    (item) => item.type === state.webhook.type
  );
});

const imSetting = computed(() => {
  const setting = settingStore.getSettingByName(Setting_SettingName.APP_IM);
  if (!setting?.value?.value) return undefined;
  const value = setting.value.value;
  if (value.case === "appIm") {
    return value.value;
  }
  return undefined;
});

const imApp = computed(() => {
  if (!selectedWebhook.value?.supportDirectMessage) {
    return undefined;
  }
  return imSetting.value?.settings.find(
    (setting) => setting.type === selectedWebhook.value?.type
  );
});

const webhookSupportDirectMessage = computed(
  () => selectedWebhook.value?.supportDirectMessage
);

const activitySupportDirectMessage = computed(() => {
  return state.webhook.notificationTypes.some(
    (event) =>
      webhookActivityItemList.value.find((item) => item.activity === event)
        ?.supportDirectMessage
  );
});

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
    name: PROJECT_V1_ROUTE_WEBHOOKS,
  });
};

const createWebhook = () => {
  useGracefulRequest(async () => {
    const { webhook } = state;
    const updatedProject = await projectWebhookV1Store.createProjectWebhook(
      props.project.name,
      webhook
    );
    projectStore.updateProjectCache({
      ...props.project,
      ...updatedProject,
    });

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
        name: PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
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
    if (state.webhook.directMessage !== props.webhook.directMessage) {
      updateMask.push("direct_message");
    }
    if (
      !isEqual(state.webhook.notificationTypes, props.webhook.notificationTypes)
    ) {
      updateMask.push("notification_type");
    }
    const updatedProject = await projectWebhookV1Store.updateProjectWebhook(
      state.webhook,
      updateMask
    );
    projectStore.updateProjectCache({
      ...props.project,
      ...updatedProject,
    });
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
    const updatedProject = await projectWebhookV1Store.deleteProjectWebhook(
      state.webhook
    );
    projectStore.updateProjectCache({
      ...props.project,
      ...updatedProject,
    });
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
