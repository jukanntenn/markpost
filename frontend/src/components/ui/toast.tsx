"use client";

import { Toast } from "@base-ui/react/toast";
import {
  CircleCheckIcon,
  InfoIcon,
  Loader2Icon,
  OctagonXIcon,
  TriangleAlertIcon,
  XIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { toastManager } from "@/stores/toast";

const iconMap: Record<string, React.ReactNode> = {
  success: <CircleCheckIcon className="size-4 text-success" />,
  error: <OctagonXIcon className="size-4 text-destructive" />,
  warning: <TriangleAlertIcon className="size-4 text-warning" />,
  info: <InfoIcon className="size-4 text-accent" />,
  loading: <Loader2Icon className="size-4 animate-spin text-muted-foreground" />,
};

function ToastList() {
  const { toasts } = Toast.useToastManager();

  return toasts.map((t) => (
    <Toast.Root
      key={t.id}
      toast={t}
      className={cn(
        "group pointer-events-auto relative flex w-full items-start gap-3 overflow-hidden rounded-lg border bg-popover p-4 shadow-lg",
        "transition-[opacity,transform] duration-200",
        "data-[starting-style]:opacity-0 data-[starting-style]:translate-y-2",
        "data-[ending-style]:opacity-0 data-[ending-style]:translate-y-2",
      )}
    >
      <Toast.Content className="flex w-full items-start gap-3">
        {iconMap[t.type ?? "info"]}
        <div className="flex-1 space-y-1">
          <Toast.Title className="text-sm font-semibold text-popover-foreground" />
          {t.description && (
            <Toast.Description className="text-xs text-muted-foreground" />
          )}
        </div>
        <Toast.Close
          className="shrink-0 rounded-md p-0.5 text-muted-foreground transition-colors hover:text-foreground"
          aria-label="Close"
        >
          <XIcon className="size-3.5" />
        </Toast.Close>
      </Toast.Content>
    </Toast.Root>
  ));
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  return (
    <Toast.Provider toastManager={toastManager} timeout={4000}>
      {children}
      <Toast.Portal>
        <Toast.Viewport
          className="fixed top-4 right-4 z-[2000] flex w-full max-w-sm flex-col gap-2 pointer-events-none"
        >
          <ToastList />
        </Toast.Viewport>
      </Toast.Portal>
    </Toast.Provider>
  );
}
