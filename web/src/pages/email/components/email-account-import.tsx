import { useMemo, useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { api, extractErrorMessage } from "@/lib/api";
import type { BatchImportGraphEmailResult } from "@/types";
import type { OutlookOAuthConfig } from "./outlook-config";

interface EmailAccountImportProps {
  config: OutlookOAuthConfig;
  onSuccess: () => void;
}

export function EmailAccountImport({ config, onSuccess }: EmailAccountImportProps) {
  const [loading, setLoading] = useState(false);
  const [text, setText] = useState("");
  const [result, setResult] = useState<BatchImportGraphEmailResult | null>(null);

  const lineCount = useMemo(() => text.split("\n").filter((line) => line.trim()).length, [text]);

  const handleImport = async () => {
    if (!text.trim()) {
      toast.error("请至少粘贴一行数据再导入。");
      return;
    }

    setLoading(true);
    setResult(null);

    try {
      const nextResult = await api.batchImportGraphEmailAccounts({
        content: text,
        default_client_id: config.clientId,
        tenant: config.tenant,
        scope: config.scope
          .trim()
          .replace(/,/g, " ")
          .split(/\s+/)
          .map((item) => item.trim())
          .filter(Boolean),
        mailbox: config.mailbox,
        graph_base_url: config.graphBaseURL,
        status: 0,
      });

      setResult(nextResult);

      if (nextResult.queued) {
        toast.success(
          `导入已入队：接受 ${nextResult.accepted} 条，跳过 ${nextResult.skipped} 条${
            nextResult.job_id ? `（job: ${nextResult.job_id}）` : ""
          }`
        );
        onSuccess();
        return;
      }

      toast.error(`未入队：接受 ${nextResult.accepted} 条，跳过 ${nextResult.skipped} 条`);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="grid gap-6">
      <Card>
        <CardHeader>
          <CardTitle>批量导入（Graph）</CardTitle>
          <CardDescription>
            后端 Graph 导入，支持以下格式：
            {" "}
            <code className="rounded bg-muted px-1 text-xs font-mono">email----refresh_token</code>
            、
            <code className="rounded bg-muted px-1 text-xs font-mono">email----client_id----refresh_token</code>
            、
            <code className="rounded bg-muted px-1 text-xs font-mono">email----password----client_id----refresh_token</code>
            。优先使用行内 <code>client_id</code>，否则用 <code>default_client_id</code>；此导入路径不发送 <code>client_secret</code>。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>账号数据</Label>
            <Textarea
              className="min-h-[240px] font-mono text-xs"
              placeholder={
                "user@outlook.com----refresh_token\nuser@outlook.com----client_id----refresh_token\nuser@outlook.com----password----client_id----refresh_token"
              }
              value={text}
              onChange={(e) => setText(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">共 {lineCount} 行</p>
          </div>

          <Button onClick={() => void handleImport()} disabled={loading || !text.trim()} className="w-full">
            {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            开始导入
          </Button>
        </CardContent>
      </Card>

      {result && (
        <Card>
          <CardHeader>
            <CardTitle>导入结果</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="mb-4 grid grid-cols-3 gap-4 text-center">
              <div className="rounded bg-muted p-2">
                <div className="text-2xl font-bold">{result.total}</div>
                <div className="text-xs text-muted-foreground">总计</div>
              </div>
              <div className="rounded bg-green-100 p-2 dark:bg-green-900">
                <div className="text-2xl font-bold text-green-600 dark:text-green-400">{result.accepted}</div>
                <div className="text-xs text-muted-foreground">接受</div>
              </div>
              <div className="rounded bg-red-100 p-2 dark:bg-red-900">
                <div className="text-2xl font-bold text-red-600 dark:text-red-400">{result.skipped}</div>
                <div className="text-xs text-muted-foreground">跳过</div>
              </div>
            </div>

            {result.queued && (
              <div className="mb-4 rounded border bg-muted p-3 text-sm text-muted-foreground">
                已入队异步导入
                {result.job_id ? `，job_id: ${result.job_id}` : ""}
                {result.task_id ? `，task_id: ${result.task_id}` : ""}。
              </div>
            )}

            {result.failures.length > 0 && (
              <div>
                <h4 className="mb-2 text-sm font-medium">失败详情</h4>
                <div className="max-h-[200px] space-y-1 overflow-y-auto rounded border bg-muted p-2 font-mono text-xs">
                  {result.failures.map((failure) => (
                    <div key={`${failure.line}-${failure.address ?? ""}`} className="text-red-500">
                      [行 {failure.line}] {failure.address || "（未知）"}: {failure.error}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
