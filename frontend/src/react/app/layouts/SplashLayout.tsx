import { Outlet, useMatches } from "react-router-dom";
import signinImage from "@/assets/illustration/signin.webp";
import signupImage from "@/assets/illustration/signup.webp";
import { AUTH_SIGNUP_MODULE } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

// Ported from `src/layouts/SplashLayout.vue`. Two-column auth chrome: an
// optional branding illustration on the left (shown unless the workspace is on
// a non-trialing enterprise plan) and the routed auth page on the right.
export function SplashLayout() {
  const matches = useMatches();
  const currentRouteName = (
    matches.at(-1)?.handle as { name?: string } | undefined
  )?.name;

  const currentPlan = useAppStore((s) => s.currentPlan());
  const isTrialing = useAppStore((s) => s.isTrialing());
  const showBrandingImage = currentPlan !== PlanType.ENTERPRISE || isTrialing;

  return (
    <div className="min-h-screen overflow-hidden flex">
      {showBrandingImage ? (
        <div className="hidden bg-white lg:block relative w-0 flex-1">
          <img
            className="absolute inset-0 h-full w-full object-cover"
            src={
              currentRouteName === AUTH_SIGNUP_MODULE
                ? signupImage
                : signinImage
            }
            alt=""
          />
        </div>
      ) : null}
      <div className="relative mx-auto flex-1 flex flex-col justify-center py-12 pb-24 px-4 sm:px-6 lg:flex-none lg:px-20 lg:w-1/2 xl:px-24">
        <Outlet />
      </div>
    </div>
  );
}
