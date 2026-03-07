import { useCallback, useEffect, useState } from "react";
import { CircleStop, Eye, Plus, RefreshCw, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Pagination } from "@/components/ui/pagination";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api, extractErrorMessage } from "@/lib/api";
import { compactId, formatDateTime } from "@/lib/format";
import type { Job } from "@/types";
import { JobCreate } from "./jobs/components/job-create";
import { JobDetails } from "./jobs/components/job-details";

const statusMap: Record<number, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  0: { label: "待执行", variant: "secondary" },
  1: { label: "执行中", variant: "default" },
  2: { label: "已成功", variant: "outline" },
  3: { label: "已失败", variant: "destructive" },
  4: { label: "已取消", variant: "secondary" }
};

export function JobsPage() {
  const [items, setItems] = useState<Job[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(true);

  // Sheet states
  const [createOpen, setCreateOpen] = useState(false);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [selectedJobId, setSelectedJobId] = useState<number | null>(null);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const res = await api.listJobs({ limit: pageSize, offset });
      setItems(res.items);
      setTotal(res.total);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [offset, pageSize]);

  const handleLimitChange = (newLimit: number) => {
    setOffset(0);
    setPageSize(newLimit);
  };

  useEffect(() => {
    void load();
  }, [load]);

  const cancelJob = async (id: number) => {
    try {
      await api.cancelJob(id);
      toast.success("任务已取消");
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const deleteJob = async (id: number) => {
    if (!confirm("确定要删除此任务记录吗？")) return;
    try {
      await api.deleteJob(id);
      toast.success("任务已删除");
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleViewDetails = (id: number) => {
    setSelectedJobId(id);
    setDetailsOpen(true);
  };

  return (
    <div className="space-y-4">
      <PageHeader title="任务" description="创建并观察异步任务，支持快速取消。">
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          创建任务
        </Button>
      </PageHeader>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div className="space-y-1">
            <CardTitle>任务列表</CardTitle>
            <CardDescription>共 {total} 条</CardDescription>
          </div>
          <Button variant="outline" size="icon" onClick={load} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
        </CardHeader>
        <CardContent>
          {items.length === 0 && !loading ? (
            <div className="rounded-lg border border-dashed border-border/80 bg-muted/25 px-4 py-8 text-center text-sm text-muted-foreground">
              暂无任务
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>动作</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>更新时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => {
                  const canCancel = item.status === 0 || item.status === 1;
                  return (
                    <TableRow key={item.id}>
                      <TableCell className="font-medium font-mono text-xs">{compactId(item.id)}</TableCell>
                      <TableCell>{item.type_key}</TableCell>
                      <TableCell>{item.action_key}</TableCell>
                      <TableCell>
                        <Badge variant={statusMap[item.status]?.variant ?? "outline"}>
                          {statusMap[item.status]?.label ?? `#${item.status}`}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-xs">{formatDateTime(item.updated_at)}</TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            size="icon"
                            variant="ghost"
                            onClick={() => handleViewDetails(item.id)}
                            title="查看详情"
                          >
                            <Eye className="h-4 w-4" />
                          </Button>
                          {canCancel && (
                            <Button
                              size="icon"
                              variant="ghost"
                              onClick={() => void cancelJob(item.id)}
                              title="取消任务"
                            >
                              <CircleStop className="h-4 w-4 text-orange-500" />
                            </Button>
                          )}
                          <Button
                            size="icon"
                            variant="ghost"
                            onClick={() => void deleteJob(item.id)}
                            title="删除任务"
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
          <Pagination total={total} limit={pageSize} offset={offset} onPageChange={setOffset} onLimitChange={handleLimitChange} />
        </CardContent>
      </Card>

      <JobCreate
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={load}
      />

      <JobDetails
        jobId={selectedJobId}
        open={detailsOpen}
        onOpenChange={setDetailsOpen}
      />
    </div>
  );
}
