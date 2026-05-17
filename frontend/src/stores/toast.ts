import { Toast } from "@base-ui/react/toast";

export const toastManager = Toast.createToastManager();

type ToastType = "success" | "error" | "info" | "warning";

function addToast(type: ToastType, message: string, options?: { description?: string }) {
  toastManager.add({
    title: message,
    description: options?.description,
    type,
  });
}

export const toast = {
  success(message: string, options?: { description?: string }) {
    addToast("success", message, options);
  },
  error(message: string, options?: { description?: string }) {
    addToast("error", message, options);
  },
  info(message: string, options?: { description?: string }) {
    addToast("info", message, options);
  },
  warning(message: string, options?: { description?: string }) {
    addToast("warning", message, options);
  },
};
