export function required(value: string): boolean {
  return value.trim().length > 0;
}

export function isoLocal(value: string): string {
  return new Date(value).toISOString();
}

export function toDateInputValue(date: Date): string {
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0, 16);
}
