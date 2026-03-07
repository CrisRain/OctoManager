import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { api, extractErrorMessage } from "@/lib/api";
import type { BatchRegisterEmailResult } from "@/types";

interface EmailAccountBatchRegisterProps {
  onSuccess: () => void;
}

export function EmailAccountBatchRegister({ onSuccess }: EmailAccountBatchRegisterProps) {
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<BatchRegisterEmailResult | null>(null);
  const [form, setForm] = useState({
    provider: "outlook",
    count: 10,
    prefix: "",
    domain: "",
    startIndex: 1,
    status: "0",
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setLoading(true);
      setResult(null);
      const res = await api.batchRegisterEmailAccounts({
        provider: form.provider,
        count: Number(form.count),
        prefix: form.prefix,
        domain: form.domain,
        start_index: Number(form.startIndex),
        status: Number(form.status),
      });
      setResult(res);
      if (res.queued) {
        toast.success(
          `批量注册已入队${res.job_id ? `（job: ${res.job_id}）` : ""}${res.task_id ? `（task: ${res.task_id}）` : ""}`
        );
      } else {
        toast.success(`批量注册完成：创建 ${res.created} 个，失败 ${res.failed} 个`);
      }
      onSuccess();
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
          <CardTitle>批量注册邮箱账号</CardTitle>
          <CardDescription>
            使用 Python 批量生成并注册邮箱账号。
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>服务商</Label>
                <Select
                  value={form.provider}
                  onValueChange={(v) => setForm({ ...form, provider: v })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="outlook">Outlook</SelectItem>
                    <SelectItem value="hotmail">Hotmail</SelectItem>
                    <SelectItem value="gmail">Gmail</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>数量</Label>
                <Input
                  type="number"
                  min={1}
                  max={200}
                  value={form.count}
                  onChange={(e) => setForm({ ...form, count: Number(e.target.value) })}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label>状态</Label>
                <Select
                  value={form.status}
                  onValueChange={(v) => setForm({ ...form, status: v })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="0">待验证 (0)</SelectItem>
                    <SelectItem value="1">已验证 (1)</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>前缀</Label>
                <Input
                  value={form.prefix}
                  onChange={(e) => setForm({ ...form, prefix: e.target.value })}
                  placeholder="user"
                />
              </div>
              <div className="space-y-2">
                <Label>域名</Label>
                <Input
                  value={form.domain}
                  onChange={(e) => setForm({ ...form, domain: e.target.value })}
                  placeholder="example.com"
                />
              </div>
              <div className="space-y-2">
                <Label>起始序号</Label>
                <Input
                  type="number"
                  min={1}
                  value={form.startIndex}
                  onChange={(e) => setForm({ ...form, startIndex: Number(e.target.value) })}
                />
              </div>
            </div>

            <Button type="submit" className="w-full" disabled={loading}>
              {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              开始批量注册
            </Button>
          </form>
        </CardContent>
      </Card>

      {result && (
        <Card>
          <CardHeader>
            <CardTitle>执行结果</CardTitle>
          </CardHeader>
          <CardContent>
            {result.queued ? (
              <div className="text-sm text-muted-foreground">
                任务已成功入队
                {result.job_id ? `，job_id: ${result.job_id}` : ""}
                {result.task_id ? `，task_id: ${result.task_id}` : ""}。
              </div>
            ) : (
              <div className="space-y-2">
                <div className="grid grid-cols-4 gap-4 text-center">
                  <div className="rounded bg-muted p-2">
                    <div className="text-2xl font-bold">{result.requested}</div>
                    <div className="text-xs text-muted-foreground">请求数</div>
                  </div>
                  <div className="rounded bg-muted p-2">
                    <div className="text-2xl font-bold">{result.generated}</div>
                    <div className="text-xs text-muted-foreground">生成数</div>
                  </div>
                  <div className="rounded bg-green-100 p-2 dark:bg-green-900">
                    <div className="text-2xl font-bold text-green-600 dark:text-green-400">
                      {result.created}
                    </div>
                    <div className="text-xs text-muted-foreground">创建数</div>
                  </div>
                  <div className="rounded bg-red-100 p-2 dark:bg-red-900">
                    <div className="text-2xl font-bold text-red-600 dark:text-red-400">
                      {result.failed}
                    </div>
                    <div className="text-xs text-muted-foreground">失败数</div>
                  </div>
                </div>

                {result.failures.length > 0 && (
                  <div className="mt-4">
                    <h4 className="mb-2 text-sm font-medium">失败详情</h4>
                    <div className="max-h-[200px] overflow-y-auto rounded border bg-muted p-2 font-mono text-xs">
                      {result.failures.map((fail, i) => (
                        <div key={i} className="mb-1 text-red-500">
                          [{fail.index}] {fail.address || "未知"}: {fail.message}
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
