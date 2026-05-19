import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";

type ButtonProps = React.ComponentProps<typeof Button>;

interface LoadingButtonProps extends ButtonProps {
  loading: boolean;
  loadingText: React.ReactNode;
}

function LoadingButton({ loading, loadingText, disabled, children, ...props }: LoadingButtonProps) {
  return (
    <Button disabled={loading || disabled} {...props}>
      {loading ? (
        <span className="inline-flex items-center gap-2">
          <Spinner className="size-4" />
          {loadingText}
        </span>
      ) : (
        children
      )}
    </Button>
  );
}

export { LoadingButton };
export type { LoadingButtonProps };
