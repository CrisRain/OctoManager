import { useCallback, useEffect, useMemo, useState } from "react";
import { BellRing, Cable, Eye, MailCheck, Shapes, Workflow } from "lucide-react";
import { toast } from "sonner";
import { PageHeader } from "@/components/page-header";
import { StatCard } from "@/components/stat-card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api, extractErrorMessage, fetchHealth } from "@/lib/api";
import { compactId, formatDateTime } from "@/lib/format";
import type { Job } from "@/types";
import { JobDetails } from "./jobs/components/job-details";

interface DashboardState {
  accountTypeCount: number;
  accountCount: number;
  emailCount: number;
  jobCount: number;
  moduleCount: number;
  health: string;
  healthTime: string;
  recentJobs: Job[];
}

const jobStatusMap: Record<number, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  0: { label: "待执行", variant: "secondary" },
  1: { label: "执行中", variant: "default" },
  2: { label: "已成功", variant: "outline" },
  3: { label: "已失败", variant: "destructive" },
  4: { label: "已取消", variant: "secondary" }
};

export function DashboardPage() {
  const [loading, setLoading] = useState(true);
  const [state, setState] = useState<DashboardState | null>(null);
  
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [selectedJobId, setSelectedJobId] = useState<number | null>(null);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const [health, accountTypes, accounts, emails, jobs, modules] = await Promise.all([
        fetchHealth(),
        api.listAccountTypes(),
        api.listAccounts(),
        api.listEmailAccounts(),
        api.listJobs({ limit: 10 }),
        api.listOctoModules()
      ]);

      // Sort jobs by updated_at desc
      const sortedJobs = jobs.items.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());

      setState({
        accountTypeCount: accountTypes.length,
        accountCount: accounts.total,
        emailCount: emails.total,
        jobCount: jobs.total,
        moduleCount: modules.length,
        health: health.status,
        healthTime: health.time,
        recentJobs: sortedJobs.slice(0, 10)
      });
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const failedJobCount = useMemo(() => {
    if (!state) {
      return 0;
    }
    return state.recentJobs.filter((job) => job.status === 3).length;
  }, [state]);

  const handleViewJob = (id: number) => {
    setSelectedJobId(id);
    setDetailsOpen(true);
  };

  return (
    <div className="space-y-4">
      <PageHeader
        title="控制台"
        description="核心实体与任务管道一屏可见，先看系统健康，再看任务状态。"
        action={
          <Button variant="outline" onClick={() => void load()} disabled={loading}>
            刷新总览
          </Button>
        }
      />

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
        {loading || !state ? (
          Array.from({ length: 5 }).map((_, idx) => <Skeleton key={idx} className="h-[132px] rounded-xl" />)
        ) : (
          <>
            <StatCard label="账号类型" value={state.accountTypeCount} hint="可注册的账号类型总数" icon={<Shapes className="h-5 w-5" />} />
            <StatCard label="账号数" value={state.accountCount} hint="资产账号池规模" icon={<Cable className="h-5 w-5" />} />
            <StatCard label="邮箱账号" value={state.emailCount} hint="邮箱账号存量" icon={<MailCheck className="h-5 w-5" />} />
            <StatCard label="任务数" value={state.jobCount} hint={`最近任务失败 ${failedJobCount}`} icon={<Workflow className="h-5 w-5" />} />
            <StatCard label="健康状态" value={state.health.toUpperCase()} hint={formatDateTime(state.healthTime)} icon={<BellRing className="h-5 w-5" />} />
          </>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>最近任务</CardTitle>
          <CardDescription>按更新时间倒序展示最近 10 条任务，直接定位失败或积压。</CardDescription>
        </CardHeader>
        <CardContent>
          {loading || !state ? (
            <div className="space-y-2">
              <Skeleton className="h-10" />
              <Skeleton className="h-10" />
              <Skeleton className="h-10" />
            </div>
          ) : state.recentJobs.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border/80 bg-muted/25 px-4 py-8 text-center text-sm text-muted-foreground">
              暂无任务记录
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>任务 ID</TableHead>
                  <TableHead>类型</TableHead>
                  <TableHead>动作</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>更新时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {state.recentJobs.map((job) => (
                  <TableRow key={job.id}>
                    <TableCell className="font-medium font-mono text-xs">{compactId(job.id)}</TableCell>
                    <TableCell>{job.type_key}</TableCell>
                    <TableCell>{job.action_key}</TableCell>
                    <TableCell>
                      <Badge variant={jobStatusMap[job.status]?.variant ?? "outline"}>
                        {jobStatusMap[job.status]?.label ?? `#${job.status}`}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-xs">{formatDateTime(job.updated_at)}</TableCell>
                    <TableCell className="text-right">
                      <Button
                        size="icon"
                        variant="ghost"
                        className="h-8 w-8"
                        onClick={() => handleViewJob(job.id)}
                      >
                        <Eye className="h-4 w-4" />
                        <span className="sr-only">查看</span>
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <JobDetails
        jobId={selectedJobId}
        open={detailsOpen}
        onOpenChange={setDetailsOpen}
      />
    </div>
  );
}
