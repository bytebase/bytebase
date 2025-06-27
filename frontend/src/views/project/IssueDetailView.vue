<template>
  <IssueReviewView />
</template>

<script lang="ts" setup>
import { usePollPlan } from "@/components/Plan";
import { IssueReviewView } from "@/components/Plan/components/IssueReviewView";

usePollPlan();
</script>
