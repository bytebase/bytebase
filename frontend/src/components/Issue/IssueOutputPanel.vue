<template>
  <h2 class="px-4 text-lg font-medium">
    Output
    <span class="text-base font-normal text-control-light">
      (these fields must be filled by the assignee before resolving the issue)
    </span>
  </h2>

  <div class="my-2 mx-4 space-y-2">
    <template v-for="(field, index) in template.outputFieldList" :key="index">
      <div class="flex flex-col space-y-1">
        <div class="textlabel">
          {{ field.name }}
          <span class="text-red-600">*</span>
          <template v-if="allowEditDatabase">
            <router-link
              :to="field.actionLink(issueContext)"
              class="ml-2 normal-link"
            >
              {{ field.actionText }}
            </router-link>
          </template>
        </div>
        <template v-if="field.type == 'String'">
          <div class="flex flex-row">
            <input
              type="text"
              class="flex-1 min-w-0 block w-full px-3 py-2 rounded-l-md border border-r border-control-border focus:mr-0.5 focus:ring-control focus:border-control sm:text-sm disabled:bg-gray-50"
              :disabled="!allowEditOutput"
              :name="field.id"
              :value="fieldValue(field)"
              autocomplete="off"
              @blur="(e: any) => updateCustomField(field, e.target.value)"
            />
            <!-- Disallow tabbing since the focus ring is partially covered by the text field due to overlaying -->
            <button
              tabindex="-1"
              :disabled="!fieldValue(field)"
              class="-ml-px px-2 py-2 border border-gray-300 text-sm font-medium text-control-light disabled:text-gray-300 bg-gray-50 hover:bg-gray-100 disabled:bg-gray-50 focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1 disabled:cursor-not-allowed"
              @click.prevent="copyText(field)"
            >
              <heroicons-outline:clipboard class="w-6 h-6" />
            </button>
            <button
              tabindex="-1"
              :disabled="!isValidLink(fieldValue(field))"
              class="-ml-px px-2 py-2 border border-gray-300 text-sm font-medium rounded-r-md text-control-light disabled:text-gray-300 bg-gray-50 hover:bg-gray-100 disabled:bg-gray-50 focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1"
              @click.prevent="goToLink(fieldValue(field))"
            >
              <heroicons-outline:external-link class="w-6 h-6" />
            </button>
          </div>
        </template>
        <div
          v-if="field.type == 'Database'"
          class="flex flex-row items-center space-x-2"
        >
          <!-- eslint-disable vue/attribute-hyphenation -->
          <DatabaseSelect
            class="mt-1 w-64"
            :disabled="!allowEditDatabase"
            :mode="'ENVIRONMENT'"
            :environmentId="environmentId"
            :selectedId="parseInt(fieldValue(field), 10) || UNKNOWN_ID"
            @select-database-id="
              (databaseId: number) => {
                trySaveCustomField(field, databaseId);
              }
            "
          />
          <template v-if="field.viewLink(issueContext)">
            <router-link
              :to="field.viewLink(issueContext)"
              class="ml-2 normal-link text-sm"
            >
              View
            </router-link>
          </template>
          <div v-if="field.resolved(issueContext)" class="text-sm text-success">
            {{ field.resolveStatusText(true) }}
          </div>
          <div v-else class="text-sm text-error">
            {{ field.resolveStatusText(false) }}
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, Ref } from "vue";
import { useRouter } from "vue-router";
import { isEqual } from "lodash-es";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import DatabaseSelect from "../DatabaseSelect.vue";
import { activeEnvironment } from "@/utils";
import { OutputField, IssueContext } from "@/plugins";
import { DatabaseId, EnvironmentId, Issue, UNKNOWN_ID } from "@/types";
import { pushNotification, useCurrentUser } from "@/store";
import { useExtraIssueLogic, useIssueLogic } from "./logic";

const router = useRouter();
const logic = useIssueLogic();
const issue = logic.issue as Ref<Issue>;
const template = logic.template;
const { allowEditOutput, updateCustomField } = useExtraIssueLogic();

const currentUser = useCurrentUser();

const environmentId = computed((): EnvironmentId => {
  return activeEnvironment(issue.value.pipeline).id;
});

const fieldValue = (field: OutputField): string => {
  return issue.value.payload[field.id];
};

const issueContext = computed((): IssueContext => {
  return {
    currentUser: currentUser.value,
    create: false,
    issue: issue.value,
  };
});

const allowEditDatabase = computed((): boolean => {
  if (!allowEditOutput.value) {
    return false;
  }
  return (
    issue.value.type == "bb.issue.database.create" ||
    issue.value.type == "bb.issue.database.grant"
  );
});

const isValidLink = (link: string): boolean => {
  return link?.trim().length > 0;
};

const copyText = (field: OutputField) => {
  toClipboard(issue.value.payload[field.id]).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: `${field.name} copied to clipboard.`,
    });
  });
};

const goToLink = (link: string) => {
  const myLink = link.trim();
  const parts = myLink.split("://");
  if (parts.length > 1) {
    window.open(myLink, "_blank");
  } else {
    if (!myLink.startsWith("/")) {
      router.push("/" + myLink);
    } else {
      router.push(myLink);
    }
  }
};

const trySaveCustomField = (field: OutputField, value: string | DatabaseId) => {
  if (!isEqual(value, fieldValue(field))) {
    updateCustomField(field, value);
  }
};
</script>
