import { Toast } from "@base-ui/react/toast";

export const toastManager = Toast.createToastManager();

type ToastType = "success" | "error" | "info" | "warning";

function createToastMethod(type: ToastType) {
  return (message: string, options?: { description?: string }) => {
    toastManager.add({ title: message, description: options?.description, type });
  };
}

export const toast = {
  success: createToastMethod("success"),
  error: createToastMethod("error"),
  info: createToastMethod("info"),
  warning: createToastMethod("warning"),
};
