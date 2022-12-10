import { useI18n } from "vue-i18n";

const instanceNamePattern = /^([0-9a-z]+-?)+[0-9a-z]+$/;

const instanceNameMinLength = 2;
const instanceNameMaxLength = 20;

export const validateInstanceName = (name: string): boolean => {
  return (
    instanceNamePattern.test(name) &&
    name.length >= instanceNameMinLength &&
    name.length <= instanceNameMaxLength
  );
};

export const getInstanceNameAttention = (): string => {
  const { t } = useI18n();

  return t("instance.instance-name-pattern", {
    min: instanceNameMinLength,
    max: instanceNameMaxLength,
  });
};
