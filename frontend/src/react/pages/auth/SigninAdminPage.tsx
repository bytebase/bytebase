import { useState } from "react";
import { useTranslation } from "react-i18next";
import { AuthFooter } from "@/react/components/auth/AuthFooter";
import { PasswordSigninForm } from "@/react/components/auth/PasswordSigninForm";
import { BytebaseLogo } from "@/react/components/BytebaseLogo";
import { useAuthStore } from "@/store";
import type { LoginRequest } from "@/types/proto-es/v1/auth_service_pb";

export function SigninAdminPage() {
  const { t } = useTranslation();
  const [isLoading, setIsLoading] = useState(false);

  const trySignin = async (request: LoginRequest) => {
    if (isLoading) return;
    setIsLoading(true);
    try {
      await useAuthStore().login({ request, redirect: true });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <div className="h-full flex flex-col justify-center mx-auto w-full max-w-sm">
        <BytebaseLogo className="mx-auto mb-8" />

        <div className="rounded-sm border border-control-border bg-white p-6">
          <p className="text-xl pl-1 font-medium mb-4">
            {t("common.sign-in-as-admin")}
          </p>
          <PasswordSigninForm
            loading={isLoading}
            showForgotPassword={false}
            onSignin={trySignin}
          />
        </div>
      </div>
      <AuthFooter />
    </>
  );
}
