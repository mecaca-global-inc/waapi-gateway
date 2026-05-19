import clsx from "clsx";

const colors: Record<string, string> = {
  STOPPED: "bg-zinc-200 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300",
  STARTING: "bg-yellow-200 text-yellow-900 dark:bg-yellow-900 dark:text-yellow-100",
  SCAN_QR: "bg-blue-200 text-blue-900 dark:bg-blue-900 dark:text-blue-100",
  WORKING: "bg-green-200 text-green-900 dark:bg-green-900 dark:text-green-100",
  FAILED: "bg-red-200 text-red-900 dark:bg-red-900 dark:text-red-100",
};

export default function Badge({ status }: { status: string }) {
  return (
    <span
      className={clsx(
        "inline-block rounded px-2 py-0.5 text-xs font-medium",
        colors[status] ?? colors.STOPPED,
      )}
    >
      {status}
    </span>
  );
}
