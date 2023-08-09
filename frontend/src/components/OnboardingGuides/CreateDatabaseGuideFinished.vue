<template>
  <teleport to="body">
    <div class="dialog-wrapper fixed top-0 left-0 w-full h-full">
      <div
        class="dialog-container relative top-1/2 -translate-y-1/2 -mt-8 mx-auto w-128 bg-white shadow-2xl rounded-lg p-4 flex flex-col justify-start items-start"
      >
        <span
          class="absolute right-3 top-3 p-px rounded cursor-pointer hover:bg-gray-100 hover:shadow"
          @click="handleCloseButtonClick"
        >
          <heroicons-outline:x class="w-6 h-auto" />
        </span>
        <p class="text-9xl w-full text-center mt-6">🎉</p>
        <div
          class="showup w-1/2 mx-auto flex flex-col justify-start items-start mt-6 text-lg leading-9"
        >
          <p>
            {{
              $t(
                "onboarding-guide.create-database-guide.finished-dialog.you-have-done"
              )
            }}
          </p>
          <p>
            ✅
            {{
              $t(
                "onboarding-guide.create-database-guide.finished-dialog.add-an-instance"
              )
            }}
          </p>
          <p>
            ✅
            {{
              $t(
                "onboarding-guide.create-database-guide.finished-dialog.create-a-project"
              )
            }}
          </p>
          <p>
            ✅
            {{
              $t(
                "onboarding-guide.create-database-guide.finished-dialog.create-an-issue"
              )
            }}
          </p>
          <p>
            ✅
            {{
              $t(
                "onboarding-guide.create-database-guide.finished-dialog.create-a-new-database"
              )
            }}
          </p>
        </div>
        <button
          class="showup self-center mt-6 mb-4 shadow text-lg px-6 py-3 rounded-md text-white bg-green-600 hover:bg-green-700"
          @click="handleCloseButtonClick"
        >
          {{
            $t(
              "onboarding-guide.create-database-guide.finished-dialog.keep-going-with-bytebase"
            )
          }}
        </button>
      </div>
    </div>
  </teleport>
</template>

<script lang="ts" setup>
import { onMounted } from "vue";
import { useOnboardingGuideStore } from "@/store";

onMounted(() => {
  const colors = ["#2563eb", "#f43f5e", "#f5a30b", "#f5e10b"];

  import("canvas-confetti").then(({ default: confetti }) => {
    const end = Date.now() + 3 * 1000;

    const framework = () => {
      confetti({
        particleCount: 4,
        angle: 60,
        spread: 60,
        origin: { x: 0 },
        zIndex: 10000,
        colors,
      });
      confetti({
        particleCount: 4,
        angle: 120,
        spread: 60,
        origin: { x: 1 },
        zIndex: 10000,
        colors,
      });

      if (Date.now() < end) {
        requestAnimationFrame(framework);
      }
    };

    framework();
  });
});

const handleCloseButtonClick = () => {
  useOnboardingGuideStore().removeGuide();
};
</script>

<style scoped>
.dialog-wrapper {
  z-index: 1000;
  background-color: rgb(33 33 33 / 50%);
}

.showup {
  animation: moveup 1s ease;
}

@keyframes moveup {
  from {
    @apply opacity-0;
    transform: translateY(24px);
  }

  to {
    @apply opacity-100;
    transform: translateY(0);
  }
}
</style>
