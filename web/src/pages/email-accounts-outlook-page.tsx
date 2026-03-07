import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { PageHeader } from "@/components/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { api, extractErrorMessage } from "@/lib/api";
import type { EmailAccount } from "@/types";
import { EmailAccountBatchRegister } from "./email/components/email-account-batch";
import { EmailAccountCreate } from "./email/components/email-account-create";
import { EmailAccountDetails } from "./email/components/email-account-details";
import { EmailAccountEdit } from "./email/components/email-account-edit";
import { EmailAccountImport } from "./email/components/email-account-import";
import { EmailAccountList } from "./email/components/email-account-list";
import { OutlookConfigPanel, useOutlookConfig } from "./email/components/outlook-config";

export function EmailAccountsOutlookPage() {
  const [items, setItems] = useState<EmailAccount[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState("list");
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [batchLoading, setBatchLoading] = useState(false);

  const [detailsOpen, setDetailsOpen] = useState(false);
  const [selectedAccountID, setSelectedAccountID] = useState<number | null>(null);

  const [editOpen, setEditOpen] = useState(false);
  const [editAccount, setEditAccount] = useState<typeof items[number] | null>(null);

  const { config, configLoading, saveConfig } = useOutlookConfig();

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const res = await api.listEmailAccounts({ limit: pageSize, offset });
      setItems(res.items);
      setTotal(res.total);
      setSelectedIds(new Set());
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [offset, pageSize]);

  const handlePageSizeChange = (newSize: number) => {
    setOffset(0);
    setPageSize(newSize);
  };

  useEffect(() => {
    void load();
  }, [load]);

  const handleVerify = async (id: number) => {
    try {
      await api.verifyEmailAccount(id);
      toast.success("邮箱账号已标记为已验证");
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm("确定要删除此邮箱账号？")) {
      return;
    }
    try {
      await api.deleteEmailAccount(id);
      toast.success("邮箱账号已删除");
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleViewDetails = (id: number) => {
    setSelectedAccountID(id);
    setDetailsOpen(true);
  };

  const handleEdit = (id: number) => {
    const account = items.find((a) => a.id === id) ?? null;
    setEditAccount(account);
    setEditOpen(true);
  };

  const handleToggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleToggleSelectAll = () => {
    if (selectedIds.size === items.length) {
      setSelectedIds(new Set());
      return;
    }
    setSelectedIds(new Set(items.map((item) => item.id)));
  };

  const handleBatchDelete = async () => {
    if (!confirm(`确定要删除选中的 ${selectedIds.size} 个邮箱账号？`)) {
      return;
    }
    setBatchLoading(true);
    try {
      const result = await api.batchDeleteEmailAccounts([...selectedIds]);
      if (result.queued) {
        toast.success(
          `批量删除已入队${result.job_id ? `（job: ${result.job_id}）` : ""}${result.task_id ? `（task: ${result.task_id}）` : ""}`
        );
      } else if (result.failed > 0) {
        toast.warning(`已删除 ${result.success} 个，失败 ${result.failed} 个`);
      } else {
        toast.success(`已删除 ${result.success} 个账号`);
      }
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setBatchLoading(false);
    }
  };

  const handleBatchVerify = async () => {
    setBatchLoading(true);
    try {
      const result = await api.batchVerifyEmailAccounts([...selectedIds]);
      if (result.queued) {
        toast.success(
          `批量验证已入队${result.job_id ? `（job: ${result.job_id}）` : ""}${result.task_id ? `（task: ${result.task_id}）` : ""}`
        );
      } else if (result.failed > 0) {
        toast.warning(`已验证 ${result.success} 个，失败 ${result.failed} 个`);
      } else {
        toast.success(`已验证 ${result.success} 个账号`);
      }
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setBatchLoading(false);
    }
  };

  return (
    <div className="space-y-4">
      <PageHeader
        title="邮箱账号 / Outlook"
        description="Outlook 账号列表、手动添加（OAuth 弹窗回调）、批量导入与批量注册。"
      />

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="w-full justify-start">
          <TabsTrigger value="list">账号列表</TabsTrigger>
          <TabsTrigger value="create">手动添加</TabsTrigger>
          <TabsTrigger value="import">批量导入</TabsTrigger>
          <TabsTrigger value="batch">批量注册</TabsTrigger>
          <TabsTrigger value="config">OAuth 配置</TabsTrigger>
        </TabsList>

        <TabsContent value="list" className="space-y-4">
          <EmailAccountList
            items={items}
            total={total}
            offset={offset}
            pageSize={pageSize}
            loading={loading}
            selectedIds={selectedIds}
            batchLoading={batchLoading}
            onVerify={handleVerify}
            onDelete={handleDelete}
            onEdit={handleEdit}
            onViewDetails={handleViewDetails}
            onPageChange={setOffset}
            onPageSizeChange={handlePageSizeChange}
            onToggleSelect={handleToggleSelect}
            onToggleSelectAll={handleToggleSelectAll}
            onBatchDelete={() => void handleBatchDelete()}
            onBatchVerify={() => void handleBatchVerify()}
            onRefresh={() => void load()}
          />
        </TabsContent>

        <TabsContent value="create">
          <EmailAccountCreate
            config={config}
            onSuccess={() => {
              void load();
              setActiveTab("list");
            }}
          />
        </TabsContent>

        <TabsContent value="import">
          <EmailAccountImport
            config={config}
            onSuccess={() => {
              void load();
            }}
          />
        </TabsContent>

        <TabsContent value="batch">
          <EmailAccountBatchRegister
            onSuccess={() => {
              void load();
              setActiveTab("list");
            }}
          />
        </TabsContent>

        <TabsContent value="config">
          <OutlookConfigPanel config={config} configLoading={configLoading} onSave={saveConfig} />
        </TabsContent>
      </Tabs>

      <EmailAccountDetails
        accountId={selectedAccountID}
        open={detailsOpen}
        onOpenChange={setDetailsOpen}
      />

      <EmailAccountEdit
        account={editAccount}
        open={editOpen}
        onOpenChange={setEditOpen}
        onSuccess={() => void load()}
      />
    </div>
  );
}
