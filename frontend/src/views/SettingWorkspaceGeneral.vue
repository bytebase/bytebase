<template>
  <div class="mt-2 space-y-6 divide-y divide-block-border">
    <div>
      <h3 class="text-lg leading-6 font-medium text-main">SQL Console URL</h3>
      <p class="mt-1 textinfolabel">
        If your team use a separate SQL console such as phpMyAdmin, you can
        provide its URL pattern here. Once provided, Bytebase will surface the
        console link on the relevant database and table UI
        <a
          href="https://docs.bytebase.com/settings/external-sql-console"
          target="_blank"
          class="normal-link"
        >
          detailed guide</a
        >.
      </p>

      <div class="mt-4">
        <input
          id="databaseURL"
          v-model="state.consoleURL"
          type="text"
          placeholder="http://phpmyadmin.example.com:8080/index.php?route=/database/sql&db={{DB_NAME}}"
          autocomplete="off"
          class="w-full textfield"
          :disabled="!allowEdit"
        />
        <div for="databaseURL" class="mt-2 textinfolabel">
          Tip: Use {{ DB_NAME_PLACEHOLDER }} as the placeholder for the actual
          database name
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
        Update
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive } from "@vue/runtime-core";
import { useStore } from "vuex";
import { isOwner } from "../utils";
import { Setting } from "../types/setting";

const DB_NAME_PLACEHOLDER = "{{DB_NAME}}";

interface LocalState {
  consoleURL: string;
}

export default {
  name: "SettingWorkspaceGeneral",
  setup() {
    const store = useStore();

    const state = reactive<LocalState>({
      consoleURL:
        store.getters["setting/settingByName"]("bb.console.url").value,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const allowEdit = computed((): boolean => {
      return isOwner(currentUser.value.role);
    });

    const allowSave = computed((): boolean => {
      return (
        state.consoleURL !=
        store.getters["setting/settingByName"]("bb.console.url").value
      );
    });

    const doSave = () => {
      if (
        state.consoleURL !=
        store.getters["setting/settingByName"]("bb.console.url").value
      ) {
        store
          .dispatch("setting/updateSettingByName", {
            name: "bb.console.url",
            value: state.consoleURL,
          })
          .then((setting: Setting) => {
            state.consoleURL = setting.value;
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
