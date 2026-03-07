import { CheckCheck, Eye, Pencil, RefreshCw, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Pagination } from "@/components/ui/pagination";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { compactId, formatDateTime } from "@/lib/format";
import type { EmailAccount } from "@/types";

interface EmailAccountListProps {
  items: EmailAccount[];
  total: number;
  offset: number;
  pageSize: number;
  loading: boolean;
  selectedIds: Set<number>;
  batchLoading: boolean;
  onVerify: (id: number) => void;
  onDelete: (id: number) => void;
  onEdit: (id: number) => void;
  onViewDetails: (id: number) => void;
  onPageChange: (offset: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  onToggleSelect: (id: number) => void;
  onToggleSelectAll: () => void;
  onBatchDelete: () => void;
  onBatchVerify: () => void;
  onRefresh: () => void;
}

export function EmailAccountList({
  items,
  total,
  offset,
  pageSize,
  loading,
  selectedIds,
  batchLoading,
  onVerify,
  onDelete,
  onEdit,
  onViewDetails,
  onPageChange,
  onPageSizeChange,
  onToggleSelect,
  onToggleSelectAll,
  onBatchDelete,
  onBatchVerify,
  onRefresh,
}: EmailAccountListProps) {
  const allSelected = items.length > 0 && selectedIds.size === items.length;
  const someSelected = selectedIds.size > 0;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <div className="space-y-1">
          <CardTitle>邮箱账号列表</CardTitle>
          <CardDescription>共 {total} 条</CardDescription>
        </div>
        <Button variant="outline" size="icon" onClick={onRefresh} disabled={loading}>
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
        </Button>
      </CardHeader>

      <div className="mx-6 mb-3 flex items-center gap-2 rounded-lg border border-border bg-muted/40 px-4 py-2 text-sm">
        <span className="text-muted-foreground">{someSelected ? `已选 ${selectedIds.size} 项` : "批量操作"}</span>
        <div className="ml-auto flex items-center gap-2">
          <Button size="sm" variant="outline" disabled={batchLoading || !someSelected} onClick={onBatchVerify}>
            <CheckCheck className="mr-1 h-3.5 w-3.5" />
            批量验证
          </Button>
          <Button size="sm" variant="destructive" disabled={batchLoading || !someSelected} onClick={onBatchDelete}>
            <Trash2 className="mr-1 h-3.5 w-3.5" />
            批量删除
          </Button>
        </div>
      </div>

      <CardContent>
        {loading ? (
          <div className="text-sm text-muted-foreground">加载中...</div>
        ) : items.length === 0 ? (
          <div className="rounded-lg border border-dashed border-border/80 bg-muted/25 px-4 py-8 text-center text-sm text-muted-foreground">
            暂无邮箱账号
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10">
                  <Checkbox
                    checked={allSelected}
                    onCheckedChange={onToggleSelectAll}
                    aria-label="全选"
                  />
                </TableHead>
                <TableHead>ID</TableHead>
                <TableHead>邮箱地址</TableHead>
                <TableHead>服务商</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>更新时间</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((item) => (
                <TableRow
                  key={item.id}
                  data-state={selectedIds.has(item.id) ? "selected" : undefined}
                >
                  <TableCell>
                    <Checkbox
                      checked={selectedIds.has(item.id)}
                      onCheckedChange={() => onToggleSelect(item.id)}
                      aria-label={`选择 ${item.address}`}
                    />
                  </TableCell>
                  <TableCell className="font-medium font-mono text-xs">
                    {compactId(item.id)}
                  </TableCell>
                  <TableCell>{item.address}</TableCell>
                  <TableCell>{item.provider ?? "-"}</TableCell>
                  <TableCell>
                    <Badge variant={item.status === 1 ? "outline" : "secondary"}>
                      {item.status === 1 ? "已验证" : "待验证"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-xs">
                    {formatDateTime(item.updated_at)}
                  </TableCell>
                  <TableCell className="text-right space-x-2">
                    <Button size="icon" variant="ghost" onClick={() => onViewDetails(item.id)} title="查看详情">
                      <Eye className="h-4 w-4" />
                    </Button>
                    <Button size="icon" variant="ghost" onClick={() => onEdit(item.id)} title="编辑">
                      <Pencil className="h-4 w-4" />
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      disabled={item.status === 1}
                      onClick={() => onVerify(item.id)}
                      title="验证"
                    >
                      <CheckCheck className="h-4 w-4" />
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      className="text-destructive hover:text-destructive"
                      onClick={() => onDelete(item.id)}
                      title="删除"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
        <Pagination total={total} limit={pageSize} offset={offset} onPageChange={onPageChange} onLimitChange={onPageSizeChange} />
      </CardContent>
    </Card>
  );
}
