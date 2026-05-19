import { cn } from "@/lib/utils";

interface PageHeadingProps {
  children: React.ReactNode;
  actions?: React.ReactNode;
  className?: string;
}

export function PageHeading({ children, actions, className }: PageHeadingProps) {
  if (actions) {
    return (
      <div className={cn("mb-6 flex items-center justify-between md:mb-8 lg:mb-12", className)}>
        <h1 className="font-display text-[28px] font-bold tracking-tight">{children}</h1>
        {actions}
      </div>
    );
  }

  return (
    <h1 className={cn("mb-6 font-display text-[28px] font-bold tracking-tight md:mb-8 lg:mb-12", className)}>
      {children}
    </h1>
  );
}
