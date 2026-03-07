import { useEffect, useState } from "react";
import { toast } from "sonner";
import { JsonView } from "@/components/json-view";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { api, extractErrorMessage } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import type { Job } from "@/types";

interface JobDetailsProps {
  jobId: number | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const statusMap: Record<number, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  0: { label: "待执行", variant: "secondary" },
  1: { label: "执行中", variant: "default" },
  2: { label: "已成功", variant: "outline" },
  3: { label: "已失败", variant: "destructive" },
  4: { label: "已取消", variant: "secondary" }
};

export function JobDetails({
  jobId,
  open,
  onOpenChange,
}: JobDetailsProps) {
  const [job, setJob] = useState<Job | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open && jobId) {
      void loadJob();
    }
  }, [open, jobId]);

  const loadJob = async () => {
    if (!jobId) return;
    try {
      setLoading(true);
      const res = await api.getJob(jobId);
      setJob(res);
    } catch (error) {
      toast.error("加载任务详情失败：" + extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="flex w-[600px] flex-col p-0 sm:max-w-[600px]">
        <SheetHeader className="border-b p-6">
          <SheetTitle>任务详情</SheetTitle>
          <SheetDescription>查看任务 {jobId} 及其最近一次执行摘要。</SheetDescription>
        </SheetHeader>

        <div className="flex-1 overflow-hidden p-6">
          <ScrollArea className="h-full pr-4">
            {loading ? (
              <div className="text-sm text-muted-foreground">加载中...</div>
            ) : job ? (
              <div className="space-y-6">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">ID</h4>
                    <p className="font-mono text-sm">{job.id}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">类型 / 动作</h4>
                    <div className="flex items-center gap-2">
                      <Badge variant="outline">{job.type_key}</Badge>
                      <span className="text-muted-foreground">/</span>
                      <Badge variant="outline">{job.action_key}</Badge>
                    </div>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">状态</h4>
                    <Badge variant={statusMap[job.status]?.variant ?? "outline"}>
                      {statusMap[job.status]?.label ?? `#${job.status}`}
                    </Badge>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">创建时间</h4>
                    <p className="text-sm">{formatDateTime(job.created_at)}</p>
                  </div>
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-muted-foreground">更新时间</h4>
                    <p className="text-sm">{formatDateTime(job.updated_at)}</p>
                  </div>
                </div>

                <div>
                  <h4 className="mb-2 text-sm font-medium text-muted-foreground">选择器</h4>
                  <div className="rounded-md border bg-muted/50 p-2">
                    <JsonView value={job.selector} />
                  </div>
                </div>

                <div>
                  <h4 className="mb-2 text-sm font-medium text-muted-foreground">参数</h4>
                  <div className="rounded-md border bg-muted/50 p-2">
                    <JsonView value={job.params} />
                  </div>
                </div>

                {job.last_run && (
                  <>
                    <div>
                      <h4 className="mb-2 text-sm font-medium text-muted-foreground">最近一次执行</h4>
                      <div className="grid grid-cols-2 gap-4 rounded-md border bg-muted/50 p-3 text-sm">
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">工作节点</h5>
                          <p className="font-mono text-xs">{job.last_run.worker_id}</p>
                        </div>
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">尝试次数</h5>
                          <p>{job.last_run.attempt}</p>
                        </div>
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">开始时间</h5>
                          <p>{formatDateTime(job.last_run.started_at)}</p>
                        </div>
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">结束时间</h5>
                          <p>{job.last_run.ended_at ? formatDateTime(job.last_run.ended_at) : "-"}</p>
                        </div>
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">错误码</h5>
                          <p>{job.last_run.error_code || "-"}</p>
                        </div>
                        <div>
                          <h5 className="mb-1 text-xs font-medium text-muted-foreground">错误信息</h5>
                          <p>{job.last_run.error_message || "-"}</p>
                        </div>
                      </div>
                    </div>

                    <div>
                      <h4 className="mb-2 text-sm font-medium text-muted-foreground">最近执行结果</h4>
                      <div className="rounded-md border bg-muted/50 p-2">
                        <JsonView value={job.last_run.result ?? {}} />
                      </div>
                    </div>
                  </>
                )}
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">任务未找到。</div>
            )}
          </ScrollArea>
        </div>
      </SheetContent>
    </Sheet>
  );
}
