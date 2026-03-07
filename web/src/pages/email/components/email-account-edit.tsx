import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { api, extractErrorMessage } from "@/lib/api";
import type { EmailAccount } from "@/types";

interface EmailAccountEditProps {
  account: EmailAccount | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function EmailAccountEdit({ account, open, onOpenChange, onSuccess }: EmailAccountEditProps) {
  const [provider, setProvider] = useState("");
  const [status, setStatus] = useState("0");
  const [graphConfig, setGraphConfig] = useState("{}");
  const [graphConfigError, setGraphConfigError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!account || !open) return;
    setProvider(account.provider ?? "");
    setStatus(String(account.status ?? 0));
    // Fetch full graph_config for editing.
    api.getEmailAccount(account.id)
      .then((full) => {
        // graph_summary is a partial view; we show graph_config as JSON.
        // The full account has graph_summary but not raw graph_config.
        // Use graph_summary fields to reconstruct a readable JSON for editing.
        if (full.graph_summary) {
          setGraphConfig(JSON.stringify(full.graph_summary, null, 2));
        } else {
          setGraphConfig("{}");
        }
      })
      .catch(() => {
        setGraphConfig("{}");
      });
    setGraphConfigError(null);
  }, [account, open]);

  function validateJson(value: string): boolean {
    try {
      JSON.parse(value);
      setGraphConfigError(null);
      return true;
    } catch {
      setGraphConfigError("无效的 JSON 格式");
      return false;
    }
  }

  async function handleSave() {
    if (!account) return;
    if (!validateJson(graphConfig)) return;

    setSaving(true);
    try {
      await api.patchEmailAccount(account.id, {
        provider: provider.trim() || undefined,
        status: Number(status) as 0 | 1,
        graph_config: JSON.parse(graphConfig),
      });
      toast.success("已保存");
      onSuccess();
      onOpenChange(false);
    } catch (e) {
      toast.error(extractErrorMessage(e));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>编辑邮箱账号</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label>邮箱地址</Label>
            <Input value={account?.address ?? ""} disabled className="text-muted-foreground" />
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-provider">服务商</Label>
            <Input
              id="edit-provider"
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
              placeholder="如: outlook, hotmail"
            />
          </div>

          <div className="space-y-2">
            <Label>状态</Label>
            <Select value={status} onValueChange={setStatus}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="0">待验证</SelectItem>
                <SelectItem value="1">已验证</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-graph-config">Graph Config (JSON)</Label>
            <textarea
              id="edit-graph-config"
              className="w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              rows={10}
              value={graphConfig}
              onChange={(e) => {
                setGraphConfig(e.target.value);
                validateJson(e.target.value);
              }}
              spellCheck={false}
            />
            {graphConfigError && (
              <p className="text-xs text-destructive">{graphConfigError}</p>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={() => void handleSave()} disabled={saving || !!graphConfigError}>
            {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
