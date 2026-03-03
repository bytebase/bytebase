<template>
  <form class="flex flex-col gap-y-6 px-1" @submit.prevent="trySignin()">
    <div>
      <label
        for="demo-account"
        class="block text-sm font-medium leading-5 text-control"
      >
        {{ $t("auth.sign-in.demo-account") }}
      </label>
      <div class="mt-1">
        <NSelect
          id="demo-account"
          v-model:value="selectedEmail"
          :options="accountOptions"
          size="large"
        />
      </div>
    </div>

    <div class="w-full">
      <NButton
        attr-type="submit"
        type="primary"
        :loading="loading"
        size="large"
        style="width: 100%"
      >
        {{ $t("common.sign-in") }}
      </NButton>
    </div>
  </form>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NButton, NSelect } from "naive-ui";
import { ref } from "vue";
import {
  type LoginRequest,
  LoginRequestSchema,
} from "@/types/proto-es/v1/auth_service_pb";

defineProps<{
  loading: boolean;
}>();

const emit = defineEmits<(event: "signin", payload: LoginRequest) => void>();

// NOSONAR: Not a real credential — fixed demo account password, only used in demo mode.
const DEMO_PASSWORD = "12345678"; // NOSONAR

const accountOptions = [
  { label: "Demo (Workspace Admin)", value: "demo@example.com" },
  { label: "Dev1 (Developer)", value: "dev1@example.com" },
  { label: "DBA1 (DBA)", value: "dba1@example.com" },
  { label: "QA1 (QA)", value: "qa1@example.com" },
];

const selectedEmail = ref("demo@example.com");

const trySignin = () => {
  emit(
    "signin",
    create(LoginRequestSchema, {
      email: selectedEmail.value,
      password: DEMO_PASSWORD,
    })
  );
};
</script>
