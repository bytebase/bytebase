<template>
  <BBModal
    class="overflow-auto"
    :title="`'${database.databaseName}' schema drift - Latest recorded schema vs Actual`"
    @close="emit('close')"
  >
    <div
      class="space-y-4 flex flex-col overflow-hidden relative"
      style="width: calc(100vw - 10rem); height: calc(100vh - 12rem)"
    >
      <DiffEditor
        v-if="!isLoading"
        class="flex-1 w-full border rounded-md overflow-clip"
        :original="originalSchema"
        :modified="modifiedSchema"
        :readonly="true"
      />
      <div v-else class="flex justify-center items-center py-10">
        <BBSpin />
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { onMounted, ref, computed } from "vue";
import { BBModal, BBSpin } from "@/bbkit";
import {
  useChangelogStore,
  useDatabaseV1Store,
  useDatabaseV1ByName,
} from "@/store";
import { Anomaly } from "@/types/proto/v1/anomaly_service";
import { ChangelogView } from "@/types/proto/v1/database_service";
import { DiffEditor } from "../MonacoEditor";

const props = defineProps<{
  anomaly: Anomaly;
}>();

const emit = defineEmits(["close"]);

const databaseStore = useDatabaseV1Store();
const changelogStore = useChangelogStore();
const isLoading = ref(true);
const originalSchema = ref<string>("");
const modifiedSchema = ref<string>("");

const { database } = useDatabaseV1ByName(
  computed(() => props.anomaly.resource)
);

onMounted(async () => {
  const database = await databaseStore.getOrFetchDatabaseByName(
    props.anomaly.resource
  );
  const changelogs = await changelogStore.getOrFetchChangelogListOfDatabase(
    database.name,
    1, // Only fetch the latest changelog.
    ChangelogView.CHANGELOG_VIEW_FULL, // Needs to fetch the full schema.
    `type = "BASELINE | MIGRATE | MIGRATE_SDL"` // Only fetch the schema related changelog.
  );
  if (changelogs.length > 0) {
    const latestChangelog = changelogs[0];
    originalSchema.value = latestChangelog.schema;
  }
  const schema = await databaseStore.fetchDatabaseSchema(
    `${database.name}/schema`
  );
  modifiedSchema.value = schema.schema;
  isLoading.value = false;
});
</script>
