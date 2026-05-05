import { Toast } from "@base-ui/react/toast";

export const toastManager = Toast.createToastManager();

export const toast = {
  success(message: string, options?: { description?: string }) {
    toastManager.add({
      title: message,
      description: options?.description,
      type: "success",
    });
  },
  error(message: string, options?: { description?: string }) {
    toastManager.add({
      title: message,
      description: options?.description,
      type: "error",
    });
  },
  info(message: string, options?: { description?: string }) {
    toastManager.add({
      title: message,
      description: options?.description,
      type: "info",
    });
  },
  warning(message: string, options?: { description?: string }) {
    toastManager.add({
      title: message,
      description: options?.description,
      type: "warning",
    });
  },
};
