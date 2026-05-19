interface PostListEmptyStateProps {
  message: string;
}

export function PostListEmptyState({ message }: PostListEmptyStateProps) {
  return (
    <p className="py-6 text-center text-sm text-muted-foreground">
      {message}
    </p>
  );
}
