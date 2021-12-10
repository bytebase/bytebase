<template>
  <div class="mt-2 space-y-6 divide-y divide-block-border">
    <div>
      <h3 class="text-lg leading-6 font-medium text-main">
        {{ $t("settings.workspace.url-section") }}
      </h3>
      <p class="mt-1 textinfolabel">
        {{ $t("settings.workspace.tip") }}
        <a
          href="https://docs.bytebase.com/settings/external-sql-console"
          target="_blank"
          class="normal-link"
        >
          {{ $t("settings.workspace.tip-link") }}
        </a>
      </p>

      <div class="mt-4">
        <input
          id="databaseUrl"
          v-model="state.consoleUrl"
          type="text"
          placeholder="http://phpmyadmin.example.com:8080/index.php?route=/database/sql&db={{DB_NAME}}"
          autocomplete="off"
          class="w-full textfield"
          :disabled="!allowEdit"
        />
        <div for="databaseUrl" class="mt-2 textinfolabel">
          {{ $t("settings.workspace.url-tip").replace('%s', placeholder) }}
        </div>
      </div>
    </div>

    <div v-if="allowEdit" class="pt-5 flex justify-end">
      <button
        type="button"
        class="btn-primary"
        :disabled="!allowSave"
        @click.prevent="doSave"
      >
        {{ $t("common.update") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useStore } from "vuex";
import { isOwner } from "../utils";
import { Setting } from "../types/setting";

const DB_NAME_PLACEHOLDER = "{{DB_NAME}}";

interface LocalState {
  consoleUrl: string;
}

export default {
  name: "SettingWorkspaceGeneral",
  data() {
    return { placeholder: "{{ DB_NAME_PLACEHOLDER }}"}
  },
  setup() {
    const store = useStore();

    const state = reactive<LocalState>({
      consoleUrl:
        store.getters["setting/settingByName"]("bb.console.url").value,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const allowEdit = computed((): boolean => {
      return isOwner(currentUser.value.role);
    });

    const allowSave = computed((): boolean => {
      return (
        state.consoleUrl !=
        store.getters["setting/settingByName"]("bb.console.url").value
      );
    });

    const doSave = () => {
      if (
        state.consoleUrl !=
        store.getters["setting/settingByName"]("bb.console.url").value
      ) {
        store
          .dispatch("setting/updateSettingByName", {
            name: "bb.console.url",
            value: state.consoleUrl,
          })
          .then((setting: Setting) => {
            state.consoleUrl = setting.value;
          });
      }
    };

    return {
      state,
      DB_NAME_PLACEHOLDER,
      allowEdit,
      allowSave,
      doSave,
    };
  },
};
</script>
