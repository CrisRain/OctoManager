import { useCallback, useEffect, useMemo, useState } from "react";
import { Play, RefreshCw, CheckCircle2, AlertCircle, Loader2, Pencil, FilePlus, Package } from "lucide-react";
import { toast } from "sonner";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Pagination } from "@/components/ui/pagination";
import { JsonView } from "@/components/json-view";
import { api, extractErrorMessage } from "@/lib/api";
import { compactId, formatDateTime } from "@/lib/format";
import type { JobRun, OctoModuleFileInfo, OctoModuleInfo, VenvInfo } from "@/types";

const HISTORY_PAGE_SIZE = 20;

export function OctoModulesPage() {
  const [modules, setModules] = useState<OctoModuleInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);

  // Dry-run form dialog state
  const [dryRunOpen, setDryRunOpen] = useState(false);
  const [dryRunTypeKey, setDryRunTypeKey] = useState("");
  const [dryRunAction, setDryRunAction] = useState("");
  const [dryRunIdentifier, setDryRunIdentifier] = useState("");
  const [dryRunSpecJson, setDryRunSpecJson] = useState("");
  const [dryRunParamsJson, setDryRunParamsJson] = useState("");
  const [dryRunRunning, setDryRunRunning] = useState(false);

  // Result dialog state
  const [resultOpen, setResultOpen] = useState(false);
  const [resultTitle, setResultTitle] = useState("");
  const [resultData, setResultData] = useState<unknown>(null);

  // Manage dialog state
  const [manageOpen, setManageOpen] = useState(false);
  const [activeTab, setActiveTab] = useState("overview");
  const [selectedModule, setSelectedModule] = useState<OctoModuleInfo | null>(null);

  // Script tab state
  const [moduleFiles, setModuleFiles] = useState<OctoModuleFileInfo[]>([]);
  const [moduleFilesLoading, setModuleFilesLoading] = useState(false);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [scriptContent, setScriptContent] = useState("");
  const [scriptLoading, setScriptLoading] = useState(false);
  const [scriptSaving, setScriptSaving] = useState(false);
  const [scriptDirty, setScriptDirty] = useState(false);
  const [newFileName, setNewFileName] = useState("");
  const [newFileOpen, setNewFileOpen] = useState(false);

  // History tab state
  const [historyItems, setHistoryItems] = useState<JobRun[]>([]);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historyOffset, setHistoryOffset] = useState(0);
  const [historyLoading, setHistoryLoading] = useState(false);

  // Dependencies tab state
  const [venvInfo, setVenvInfo] = useState<VenvInfo | null>(null);
  const [venvLoading, setVenvLoading] = useState(false);
  const [reqContent, setReqContent] = useState("");
  const [reqDirty, setReqDirty] = useState(false);
  const [manualPkgs, setManualPkgs] = useState("");
  const [installing, setInstalling] = useState(false);
  const [installOutput, setInstallOutput] = useState("");

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const res = await api.listOctoModules();
      setModules(res);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const handleSync = async () => {
    try {
      setSyncing(true);
      const res = await api.syncOctoModules();
      toast.success(`同步完成：新建 ${res.created}，已存在 ${res.existing}，失败 ${res.failed}`);
      await load();
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setSyncing(false);
    }
  };

  const openDryRunForm = (typeKey: string) => {
    setDryRunTypeKey(typeKey);
    setDryRunAction("");
    setDryRunIdentifier("");
    setDryRunSpecJson("");
    setDryRunParamsJson("");
    setDryRunOpen(true);
  };

  const handleDryRun = async () => {
    if (!dryRunAction.trim()) {
      toast.error("动作不能为空");
      return;
    }
    if (!dryRunIdentifier.trim()) {
      toast.error("账号标识符不能为空");
      return;
    }

    let spec: Record<string, unknown> | undefined;
    let params: Record<string, unknown> | undefined;
    try {
      if (dryRunSpecJson.trim()) spec = JSON.parse(dryRunSpecJson) as Record<string, unknown>;
    } catch {
      toast.error("账号 Spec JSON 格式无效");
      return;
    }
    try {
      if (dryRunParamsJson.trim()) params = JSON.parse(dryRunParamsJson) as Record<string, unknown>;
    } catch {
      toast.error("参数 JSON 格式无效");
      return;
    }

    setDryRunRunning(true);
    try {
      const res = await api.dryRunOctoModule(dryRunTypeKey, {
        action: dryRunAction.trim(),
        account: { identifier: dryRunIdentifier.trim(), ...(spec ? { spec } : {}) },
        ...(params ? { params } : {}),
      });
      setDryRunOpen(false);
      setResultTitle(`Dry Run: ${dryRunTypeKey} / ${dryRunAction}`);
      setResultData(res);
      setResultOpen(true);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setDryRunRunning(false);
    }
  };

  const handleEnsure = async (typeKey: string) => {
    try {
      const res = await api.ensureOctoModule(typeKey);
      setResultTitle(`Ensure: ${typeKey}`);
      setResultData(res);
      setResultOpen(true);
      toast.success("Ensure 完成");
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const openManage = (module: OctoModuleInfo) => {
    setSelectedModule(module);
    setActiveTab("overview");
    setManageOpen(true);
  };

  // Load script files when script tab is active
  useEffect(() => {
    if (!manageOpen || !selectedModule) return;
    if (activeTab !== "script") return;
    let mounted = true;
    setModuleFilesLoading(true);
    setModuleFiles([]);
    setSelectedFile(null);
    setScriptContent("");
    setScriptDirty(false);
    api
      .listOctoModuleFiles(selectedModule.type_key)
      .then((res) => {
        if (!mounted) return;
        setModuleFiles(res.files);
        const entry = res.entry_file || "main.py";
        const hasEntry = res.files.some((f) => f.name === entry);
        const first = hasEntry ? entry : (res.files[0]?.name ?? null);
        setSelectedFile(first);
      })
      .catch((error) => {
        toast.error(extractErrorMessage(error));
      })
      .finally(() => {
        if (mounted) setModuleFilesLoading(false);
      });
    return () => {
      mounted = false;
    };
  }, [manageOpen, selectedModule, activeTab]);

  // Load file content when a file is selected
  useEffect(() => {
    if (!manageOpen || !selectedModule || !selectedFile) return;
    if (activeTab !== "script") return;
    let mounted = true;
    setScriptLoading(true);
    api
      .getOctoModuleFile(selectedModule.type_key, selectedFile)
      .then((res) => {
        if (mounted) {
          setScriptContent(res.content ?? "");
          setScriptDirty(false);
        }
      })
      .catch((error) => {
        toast.error(extractErrorMessage(error));
      })
      .finally(() => {
        if (mounted) setScriptLoading(false);
      });
    return () => {
      mounted = false;
    };
  }, [manageOpen, selectedModule, activeTab, selectedFile]);

  // Load run history when history tab is active
  useEffect(() => {
    if (!manageOpen || !selectedModule) return;
    if (activeTab !== "history") return;
    let mounted = true;
    setHistoryLoading(true);
    api
      .listOctoModuleRuns(selectedModule.type_key, { limit: HISTORY_PAGE_SIZE, offset: historyOffset })
      .then((res) => {
        if (mounted) {
          setHistoryItems(res.items);
          setHistoryTotal(res.total);
        }
      })
      .catch((error) => {
        toast.error(extractErrorMessage(error));
      })
      .finally(() => {
        if (mounted) setHistoryLoading(false);
      });
    return () => {
      mounted = false;
    };
  }, [manageOpen, selectedModule, activeTab, historyOffset]);

  // Load venv info when deps tab is active
  useEffect(() => {
    if (!manageOpen || !selectedModule) return;
    if (activeTab !== "deps") return;
    let mounted = true;
    setVenvLoading(true);
    setVenvInfo(null);
    setInstallOutput("");
    api
      .getModuleVenv(selectedModule.type_key)
      .then((res) => {
        if (!mounted) return;
        setVenvInfo(res);
        setReqContent(res.requirements_content ?? "");
        setReqDirty(false);
      })
      .catch((error) => {
        toast.error(extractErrorMessage(error));
      })
      .finally(() => {
        if (mounted) setVenvLoading(false);
      });
    return () => {
      mounted = false;
    };
  }, [manageOpen, selectedModule, activeTab]);

  useEffect(() => {
    setHistoryOffset(0);
  }, [selectedModule?.type_key, activeTab]);

  const saveScript = async () => {
    if (!selectedModule || !selectedFile) return;
    setScriptSaving(true);
    try {
      await api.updateOctoModuleFile(selectedModule.type_key, selectedFile, { content: scriptContent });
      toast.success("文件已保存");
      setScriptDirty(false);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setScriptSaving(false);
    }
  };

  const createNewFile = async () => {
    const name = newFileName.trim();
    if (!name || !selectedModule) return;
    try {
      await api.updateOctoModuleFile(selectedModule.type_key, name, { content: "" });
      toast.success(`文件 ${name} 已创建`);
      setNewFileName("");
      setNewFileOpen(false);
      const res = await api.listOctoModuleFiles(selectedModule.type_key);
      setModuleFiles(res.files);
      setSelectedFile(name);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    }
  };

  const handleInstallFromReqs = async () => {
    if (!selectedModule) return;
    setInstalling(true);
    setInstallOutput("");
    try {
      const res = await api.installModuleDeps(selectedModule.type_key, {
        from_requirements: true,
        requirements_content: reqDirty ? reqContent : undefined,
      });
      setInstallOutput(res.output);
      if (res.success) {
        toast.success("依赖安装成功");
        setReqDirty(false);
      } else {
        toast.error("安装失败，请查看输出");
      }
      // Refresh venv info
      const venv = await api.getModuleVenv(selectedModule.type_key);
      setVenvInfo(venv);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setInstalling(false);
    }
  };

  const handleInstallPackages = async () => {
    if (!selectedModule) return;
    const pkgs = manualPkgs
      .split(/[\n,]+/)
      .map((p) => p.trim())
      .filter(Boolean);
    if (pkgs.length === 0) {
      toast.error("请输入至少一个包名");
      return;
    }
    setInstalling(true);
    setInstallOutput("");
    try {
      const res = await api.installModuleDeps(selectedModule.type_key, { packages: pkgs });
      setInstallOutput(res.output);
      if (res.success) {
        toast.success("包安装成功");
        setManualPkgs("");
      } else {
        toast.error("安装失败，请查看输出");
      }
      const venv = await api.getModuleVenv(selectedModule.type_key);
      setVenvInfo(venv);
    } catch (error) {
      toast.error(extractErrorMessage(error));
    } finally {
      setInstalling(false);
    }
  };

  const statusBadge = useMemo(() => {
    if (!selectedModule) return null;
    if (selectedModule.exists) {
      return (
        <Badge variant="outline" className="text-green-600 border-green-200 bg-green-50">
          <CheckCircle2 className="mr-1 h-3 w-3" />
          Ready
        </Badge>
      );
    }
    if (selectedModule.error) {
      return (
        <Badge variant="destructive">
          <AlertCircle className="mr-1 h-3 w-3" />
          Error
        </Badge>
      );
    }
    return <Badge variant="secondary">Unknown</Badge>;
  }, [selectedModule]);

  return (
    <div className="space-y-4">
      <PageHeader title="Octo 模块" description="管理 Octo 模块，运行诊断并编辑模块资产。">
        <Button onClick={handleSync} disabled={syncing}>
          <RefreshCw className={`mr-2 h-4 w-4 ${syncing ? "animate-spin" : ""}`} />
          同步模块
        </Button>
      </PageHeader>

      <Card>
        <CardHeader>
          <CardTitle>模块列表</CardTitle>
          <CardDescription>共 {modules.length} 个</CardDescription>
        </CardHeader>
        <CardContent>
          {modules.length === 0 && !loading ? (
            <div className="rounded-lg border border-dashed border-border/80 bg-muted/25 px-4 py-8 text-center text-sm text-muted-foreground">
              暂无模块，请先点击"同步模块"。
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>类型键</TableHead>
                  <TableHead>分类</TableHead>
                  <TableHead>来源</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {modules.map((module) => (
                  <TableRow key={module.type_key}>
                    <TableCell className="font-medium font-mono">{module.type_key}</TableCell>
                    <TableCell>{module.category}</TableCell>
                    <TableCell className="text-muted-foreground text-xs">{module.source}</TableCell>
                    <TableCell>
                      {module.exists ? (
                        <Badge variant="outline" className="text-green-600 border-green-200 bg-green-50">
                          <CheckCircle2 className="mr-1 h-3 w-3" />
                          Ready
                        </Badge>
                      ) : module.error ? (
                        <Badge variant="destructive">
                          <AlertCircle className="mr-1 h-3 w-3" />
                          Error
                        </Badge>
                      ) : (
                        <Badge variant="secondary">Unknown</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => openManage(module)}
                          title="Manage"
                        >
                          <Pencil className="mr-1 h-3 w-3" />
                          管理
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => openDryRunForm(module.type_key)}
                          title="试运行"
                        >
                          <Play className="mr-1 h-3 w-3" />
                          试运行
                        </Button>
                        <Button
                          size="sm"
                          onClick={() => void handleEnsure(module.type_key)}
                          title="确保初始化"
                        >
                          <CheckCircle2 className="mr-1 h-3 w-3" />
                          初始化
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={manageOpen} onOpenChange={setManageOpen}>
        <DialogContent className="max-w-5xl max-h-[80vh] flex flex-col">
          <DialogHeader>
            <DialogTitle>
              {selectedModule ? `管理 ${selectedModule.type_key}` : "管理模块"}
            </DialogTitle>
            <DialogDescription>查看并编辑模块脚本、依赖与执行历史。</DialogDescription>
          </DialogHeader>

          <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 overflow-hidden">
            <TabsList>
              <TabsTrigger value="overview">概览</TabsTrigger>
              <TabsTrigger value="script">脚本</TabsTrigger>
              <TabsTrigger value="deps">依赖</TabsTrigger>
              <TabsTrigger value="history">执行历史</TabsTrigger>
            </TabsList>

            <TabsContent value="overview" className="mt-4">
              {selectedModule ? (
                <div className="space-y-4">
                  <div className="flex flex-wrap items-center gap-3">
                    <Badge variant="secondary">{selectedModule.category}</Badge>
                    {statusBadge}
                    <span className="text-xs text-muted-foreground">{selectedModule.source}</span>
                  </div>
                  <div className="rounded-md border bg-muted/40 p-4 space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">类型键</span>
                      <span className="font-mono">{selectedModule.type_key}</span>
                    </div>
                    <div className="flex justify-between gap-4">
                      <span className="text-muted-foreground shrink-0">模块目录</span>
                      <span className="font-mono text-xs text-right break-all">{selectedModule.module_dir || "-"}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">入口文件</span>
                      <span className="font-mono text-xs">{selectedModule.entry_file || "-"}</span>
                    </div>
                  </div>
                  {selectedModule.error ? (
                    <div className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">
                      {selectedModule.error}
                    </div>
                  ) : null}
                </div>
              ) : null}
            </TabsContent>

            <TabsContent value="script" className="mt-4">
              {moduleFilesLoading ? (
                <div className="text-sm text-muted-foreground">加载文件列表...</div>
              ) : (
                <div className="flex gap-3 h-[380px]">
                  {/* File list sidebar */}
                  <div className="w-44 shrink-0 flex flex-col gap-1 border rounded-md p-2 overflow-y-auto">
                    {moduleFiles.length === 0 ? (
                      <div className="text-xs text-muted-foreground text-center py-4">暂无文件</div>
                    ) : (
                      moduleFiles.map((f) => (
                        <button
                          key={f.name}
                          onClick={() => {
                            if (selectedFile !== f.name) {
                              setSelectedFile(f.name);
                              setScriptDirty(false);
                            }
                          }}
                          className={`text-left text-xs px-2 py-1.5 rounded truncate flex items-center gap-1 ${
                            selectedFile === f.name
                              ? "bg-primary text-primary-foreground"
                              : "hover:bg-muted"
                          }`}
                        >
                          {f.is_entry && <CheckCircle2 className="h-3 w-3 shrink-0 opacity-70" />}
                          <span className="truncate font-mono">{f.name}</span>
                        </button>
                      ))
                    )}
                    <div className="mt-auto pt-2 border-t">
                      {newFileOpen ? (
                        <div className="flex flex-col gap-1">
                          <Input
                            className="h-6 text-xs"
                            placeholder="文件名"
                            value={newFileName}
                            onChange={(e) => setNewFileName(e.target.value)}
                            onKeyDown={(e) => { if (e.key === "Enter") void createNewFile(); }}
                            autoFocus
                          />
                          <div className="flex gap-1">
                            <Button size="sm" className="h-6 text-xs flex-1" onClick={() => void createNewFile()}>确定</Button>
                            <Button size="sm" variant="ghost" className="h-6 text-xs" onClick={() => { setNewFileOpen(false); setNewFileName(""); }}>取消</Button>
                          </div>
                        </div>
                      ) : (
                        <Button size="sm" variant="ghost" className="w-full h-6 text-xs" onClick={() => setNewFileOpen(true)}>
                          <FilePlus className="mr-1 h-3 w-3" />
                          新建文件
                        </Button>
                      )}
                    </div>
                  </div>
                  {/* Editor */}
                  <div className="flex-1 flex flex-col gap-2 min-w-0">
                    <div className="flex items-center justify-between">
                      <span className="text-xs text-muted-foreground font-mono">{selectedFile ?? "—"}</span>
                      <Button size="sm" onClick={() => void saveScript()} disabled={!scriptDirty || scriptSaving || !selectedFile}>
                        {scriptSaving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                        保存
                      </Button>
                    </div>
                    {scriptLoading ? (
                      <div className="text-sm text-muted-foreground">加载中...</div>
                    ) : selectedFile ? (
                      <Textarea
                        className="font-mono text-xs flex-1 resize-none"
                        value={scriptContent}
                        onChange={(e) => {
                          setScriptContent(e.target.value);
                          setScriptDirty(true);
                        }}
                      />
                    ) : (
                      <div className="text-sm text-muted-foreground">请从左侧选择文件</div>
                    )}
                  </div>
                </div>
              )}
            </TabsContent>

            <TabsContent value="deps" className="mt-4 space-y-4 overflow-y-auto max-h-[480px]">
              {venvLoading ? (
                <div className="text-sm text-muted-foreground flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  加载中...
                </div>
              ) : venvInfo ? (
                <>
                  {/* Venv status */}
                  <div className="rounded-md border bg-muted/40 p-4 space-y-2 text-sm">
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">虚拟环境</span>
                      {venvInfo.exists ? (
                        <Badge variant="outline" className="text-green-600 border-green-200 bg-green-50">
                          <CheckCircle2 className="mr-1 h-3 w-3" />
                          已创建
                        </Badge>
                      ) : (
                        <Badge variant="secondary">未创建（首次安装时自动创建）</Badge>
                      )}
                    </div>
                    <div className="flex justify-between gap-4">
                      <span className="text-muted-foreground shrink-0">Python 路径</span>
                      <span className="font-mono text-xs text-right break-all text-muted-foreground">{venvInfo.python_path}</span>
                    </div>
                    <div className="flex justify-between gap-4">
                      <span className="text-muted-foreground shrink-0">Venv 目录</span>
                      <span className="font-mono text-xs text-right break-all text-muted-foreground">{venvInfo.dir}</span>
                    </div>
                  </div>

                  {/* requirements.txt */}
                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <Label className="flex items-center gap-1.5">
                        <Package className="h-3.5 w-3.5" />
                        requirements.txt
                        {reqDirty && <span className="text-xs text-muted-foreground">（已修改）</span>}
                      </Label>
                      <Button
                        size="sm"
                        onClick={() => void handleInstallFromReqs()}
                        disabled={installing}
                      >
                        {installing && <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />}
                        安装 requirements.txt
                      </Button>
                    </div>
                    <Textarea
                      className="font-mono text-xs min-h-[120px]"
                      placeholder="# 每行一个包，例如：&#10;requests==2.31.0&#10;beautifulsoup4"
                      value={reqContent}
                      onChange={(e) => {
                        setReqContent(e.target.value);
                        setReqDirty(true);
                      }}
                    />
                    {!venvInfo.has_requirements && !reqDirty && (
                      <p className="text-xs text-muted-foreground">暂无 requirements.txt，可在此编辑后安装。</p>
                    )}
                  </div>

                  {/* Manual packages */}
                  <div className="space-y-2">
                    <Label>手动安装包</Label>
                    <div className="flex gap-2">
                      <Input
                        className="font-mono text-xs"
                        placeholder="requests, beautifulsoup4==4.12.0"
                        value={manualPkgs}
                        onChange={(e) => setManualPkgs(e.target.value)}
                        onKeyDown={(e) => { if (e.key === "Enter") void handleInstallPackages(); }}
                      />
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => void handleInstallPackages()}
                        disabled={installing}
                      >
                        {installing && <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />}
                        安装
                      </Button>
                    </div>
                    <p className="text-xs text-muted-foreground">多个包用逗号或换行分隔。</p>
                  </div>

                  {/* Install output */}
                  {installOutput && (
                    <div className="space-y-1">
                      <Label className="text-xs">安装输出</Label>
                      <ScrollArea className="h-40 w-full rounded-md border bg-muted/50 p-3">
                        <pre className="text-xs font-mono whitespace-pre-wrap break-all">{installOutput}</pre>
                      </ScrollArea>
                    </div>
                  )}
                </>
              ) : null}
            </TabsContent>

            <TabsContent value="history" className="mt-4">
              <div className="mb-3 text-sm text-muted-foreground">该模块的执行历史记录。</div>
              {historyLoading ? (
                <div className="text-sm text-muted-foreground">加载历史中...</div>
              ) : historyItems.length === 0 ? (
                <div className="rounded-lg border border-dashed border-border/80 bg-muted/25 px-4 py-8 text-center text-sm text-muted-foreground">
                  暂无执行记录。
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>任务</TableHead>
                      <TableHead>动作</TableHead>
                      <TableHead>尝试次数</TableHead>
                      <TableHead>账号</TableHead>
                      <TableHead>开始时间</TableHead>
                      <TableHead>结束时间</TableHead>
                      <TableHead>错误信息</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {historyItems.map((run) => (
                      <TableRow key={run.id}>
                        <TableCell className="font-mono text-xs">{compactId(run.job_id)}</TableCell>
                        <TableCell>{run.job_action_key}</TableCell>
                        <TableCell>{run.attempt}</TableCell>
                        <TableCell className="font-mono text-xs">{run.account_id ? compactId(run.account_id) : "-"}</TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {formatDateTime(run.started_at)}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {run.ended_at ? formatDateTime(run.ended_at) : "-"}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {run.error_message || "-"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
              <Pagination
                total={historyTotal}
                limit={HISTORY_PAGE_SIZE}
                offset={historyOffset}
                onPageChange={setHistoryOffset}
              />
            </TabsContent>
          </Tabs>
        </DialogContent>
      </Dialog>

      <Dialog open={dryRunOpen} onOpenChange={setDryRunOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>试运行：{dryRunTypeKey}</DialogTitle>
            <DialogDescription>填写参数后执行试运行。</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <Label>动作 <span className="text-destructive">*</span></Label>
              <Input
                placeholder="例如 VERIFY"
                value={dryRunAction}
                onChange={(e) => setDryRunAction(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>账号标识符 <span className="text-destructive">*</span></Label>
              <Input
                placeholder="账号标识符"
                value={dryRunIdentifier}
                onChange={(e) => setDryRunIdentifier(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>账号 Spec <span className="text-xs text-muted-foreground">（可选 JSON）</span></Label>
              <Textarea
                className="font-mono text-xs min-h-[72px]"
                placeholder="{}"
                value={dryRunSpecJson}
                onChange={(e) => setDryRunSpecJson(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>参数 <span className="text-xs text-muted-foreground">（可选 JSON）</span></Label>
              <Textarea
                className="font-mono text-xs min-h-[72px]"
                placeholder="{}"
                value={dryRunParamsJson}
                onChange={(e) => setDryRunParamsJson(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDryRunOpen(false)}>取消</Button>
            <Button onClick={() => void handleDryRun()} disabled={dryRunRunning}>
              {dryRunRunning && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              <Play className="mr-1 h-3 w-3" />
              执行
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={resultOpen} onOpenChange={setResultOpen}>
        <DialogContent className="max-w-3xl max-h-[80vh] flex flex-col">
          <DialogHeader>
            <DialogTitle>{resultTitle}</DialogTitle>
            <DialogDescription>执行结果详情</DialogDescription>
          </DialogHeader>
          <div className="flex-1 overflow-hidden border rounded-md bg-muted/50 p-4 mt-2">
            <ScrollArea className="h-full">
              <JsonView value={resultData} />
            </ScrollArea>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
