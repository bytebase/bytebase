<template>
  <form
    class="space-y-6 divide-y divide-block-border"
    @submit.prevent="$emit('create', state.backupName)"
  >
    <div class="space-y-4">
      <div class="grid grid-cols-3 gap-y-6 gap-x-4">
        <div class="col-span-3">
          <label for="name" class="textlabel">
            Backup name <span class="text-red-600">*</span>
          </label>
          <input
            id="name"
            v-model="state.backupName"
            required
            name="name"
            type="text"
            class="textfield mt-1 w-full"
          />
        </div>
      </div>
    </div>
    <!-- Create button group -->
    <div class="pt-4 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="$emit('cancel')"
      >
        Cancel
      </button>
      <button
        type="submit"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
      >
        Create backup
      </button>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, PropType, reactive } from "vue";
import { Database } from "../types";
import { isEmpty } from "lodash";
import moment from "moment";
import slug from "slug";

interface LocalState {
  backupName: string;
}

export default {
  name: "DatabaseBackupCreateForm",
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
  },
  emits: ["create", "cancel"],
  setup(props) {
    const state = reactive<LocalState>({
      // The default format is consistent with the default automatic backup name format used in the server.
      backupName: `${slug(props.database.project.name)}-${slug(
        props.database.instance.environment.name
      )}-${moment.utc().local().format("YYYYMMDDTHHmmss")}`,
    });

    const allowCreate = computed(() => {
      return !isEmpty(state.backupName);
    });

    return {
      state,
      allowCreate,
    };
  },
};
</script>
