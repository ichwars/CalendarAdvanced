export function EmptyState({ message }: { message: string }) {
  return <p className="empty" role="status">{message}</p>;
}
