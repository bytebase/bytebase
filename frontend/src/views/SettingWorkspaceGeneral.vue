<template>
  <div class="mt-2 space-y-8 divide-y divide-gray-200">
    <div>
      <h3 class="text-lg leading-6 font-medium text-main">SQL Console</h3>
      <p class="mt-1 text-sm text-control-light">
        If your team use a separate SQL console such as phpMyAdmin, you can
        provide its URL pattern here, so that Bytebase can surface the console
        link on the relevant database and table UI.
        <a
          href="https://docs.bytebase.com/settings/external-sql-console"
          target="_blank"
          class="normal-link"
        >
          detailed guide</a
        >
      </p>

      <div class="mt-6 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
        <div class="sm:col-span-6">
          <label for="databaseURL" class="textlabel"> Database URL </label>
          <label for="databaseURL" class="textinfolabel">
            use {{ DB_NAME_PLACEHOLDER }} as the placeholder for the actual
            database name
          </label>
          <div class="mt-1">
            <input
              type="text"
              id="databaseURL"
              placeholder="http://phpmyadmin.example.com:8080/index.php?route=/database/sql&db={{DB_NAME}}"
              autocomplete="off"
              class="w-full textfield"
              :disabled="!allowEdit"
              v-model="state.databaseConsoleURL"
            />
          </div>
        </div>
      </div>
      <div class="mt-6 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
        <div class="sm:col-span-6">
          <label for="tableURL" class="textlabel"> Table URL </label>
          <label for="databaseURL" class="textinfolabel">
            use {{ DB_NAME_PLACEHOLDER }} and {{ TABLE_NAME_PLACEHOLDER }} as
            the placeholder for the actual database and table name
          </label>
          <div class="mt-1">
            <input
              type="text"
              id="tableURL"
              placeholder="http://phpmyadmin.example.com:8080/index.php?route=/table/sql&db={{DB_NAME}}&table={{TABLE_NAME}}"
              autocomplete="off"
              class="w-full textfield"
              :disabled="!allowEdit"
              v-model="state.tableConsoleURL"
            />
          </div>
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
const TABLE_NAME_PLACEHOLDER = "{{TABLE_NAME}}";

interface LocalState {
  databaseConsoleURL: string;
  tableConsoleURL: string;
}

export default {
  name: "SettingWorkspaceGeneral",
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      databaseConsoleURL: store.getters["setting/settingByName"](
        "bb.console.database"
      ).value,
      tableConsoleURL:
        store.getters["setting/settingByName"]("bb.console.table").value,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const allowEdit = computed((): boolean => {
      return isOwner(currentUser.value.role);
    });

    const allowSave = computed((): boolean => {
      return (
        state.databaseConsoleURL !=
          store.getters["setting/settingByName"]("bb.console.database").value ||
        state.tableConsoleURL !=
          store.getters["setting/settingByName"]("bb.console.table").value
      );
    });

    const doSave = () => {
      if (
        state.databaseConsoleURL !=
        store.getters["setting/settingByName"]("bb.console.database").value
      ) {
        store
          .dispatch("setting/updateSettingByName", {
            name: "bb.console.database",
            value: state.databaseConsoleURL,
          })
          .then((setting: Setting) => {
            state.databaseConsoleURL = setting.value;
          });
      }

      if (
        state.tableConsoleURL !=
        store.getters["setting/settingByName"]("bb.console.table").value
      ) {
        store
          .dispatch("setting/updateSettingByName", {
            name: "bb.console.table",
            value: state.tableConsoleURL,
          })
          .then((setting: Setting) => {
            state.tableConsoleURL = setting.value;
          });
      }
    };

    return {
      state,
      DB_NAME_PLACEHOLDER,
      TABLE_NAME_PLACEHOLDER,
      allowEdit,
      allowSave,
      doSave,
    };
  },
};
</script>
