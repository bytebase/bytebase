<template>
  <form
    class="space-y-6 divide-y divide-block-border"
    @submit.prevent="$emit('create', state.backupName)"
  >
    <div class="space-y-4">
      <div class="grid grid-cols-3 gap-y-6 gap-x-4">
        <div class="col-span-3">
          <label for="name" class="textlabel">
            {{ $t("database.backup-name") }} <span class="text-red-600">*</span>
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
        {{ $t("common.cancel") }}
      </button>
      <button
        type="submit"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
      >
        {{ $t("database.create-backup") }}
      </button>
    </div>
  </form>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import { isEmpty } from "lodash-es";
import slug from "slug";
import { computed, PropType, reactive } from "vue";
import { ComposedDatabase } from "../types";

dayjs.extend(utc);

interface LocalState {
  backupName: string;
}

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
});

defineEmits(["create", "cancel"]);

const buildBackupName = (database: ComposedDatabase) => {
  // The default format is consistent with the default automatic backup name format used in the server.
  return [
    slug(props.database.projectEntity.title),
    slug(props.database.instanceEntity.environmentEntity.title),
    dayjs.utc().local().format("YYYYMMDDTHHmmss"),
  ].join("-");
};

const state = reactive<LocalState>({
  backupName: buildBackupName(props.database),
});

const allowCreate = computed(() => {
  return !isEmpty(state.backupName);
});
</script>
