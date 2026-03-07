import { prettyJSON } from "@/lib/format";
import { cn } from "@/lib/utils";

interface JsonViewProps {
  value: unknown;
  className?: string;
}

export function JsonView({ value, className }: JsonViewProps) {
  return (
    <pre
      className={cn(
        "max-h-44 overflow-auto rounded-lg border border-border/80 bg-muted/35 p-3 font-mono text-xs leading-5",
        className
      )}
    >
      {prettyJSON(value)}
    </pre>
  );
}
