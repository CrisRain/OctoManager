import { useEffect, useState } from "react";
import { Loader2, Mail, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Pagination } from "@/components/ui/pagination";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { api, extractErrorMessage } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import type { EmailAccount, EmailMessageDetail, EmailMessageSummary } from "@/types";

interface EmailAccountDetailsProps {
  accountId: number | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface MessageDetailDialogProps {
  detail: EmailMessageDetail | null;
  loading: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const MESSAGE_PAGE_SIZE = 10;

function MessageDetailDialog({ detail, loading, open, onOpenChange }: MessageDetailDialogProps) {
  const [activeBodyTab, setActiveBodyTab] = useState<"html" | "text">("html");

  useEffect(() => {
    if (detail) {
      setActiveBodyTab(detail.html_body ? "html" : "text");
    }
  }, [detail]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[80vh] max-w-3xl flex-col p-0">
        <DialogHeader className="shrink-0 border-b px-6 pb-4 pt-6">
          <DialogTitle className="pr-8 text-base leading-snug">
            {loading ? "加载中..." : detail?.subject || "(无主题)"}
          </DialogTitle>
        </DialogHeader>

        {loading ? (
          <div className="flex flex-1 items-center justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : detail ? (
          <div className="flex flex-1 flex-col gap-3 overflow-hidden px-6 pb-6 pt-4">
            <div className="shrink-0 space-y-1 text-sm text-muted-foreground">
              <div className="flex gap-2">
                <span className="w-14 shrink-0 font-medium text-foreground">发件人</span>
                <span className="break-all">{detail.from}</span>
              </div>
              {detail.to && (
                <div className="flex gap-2">
                  <span className="w-14 shrink-0 font-medium text-foreground">收件人</span>
                  <span className="break-all">{detail.to}</span>
                </div>
              )}
              {detail.cc && (
                <div className="flex gap-2">
                  <span className="w-14 shrink-0 font-medium text-foreground">抄送</span>
                  <span className="break-all">{detail.cc}</span>
                </div>
              )}
              <div className="flex gap-2">
                <span className="w-14 shrink-0 font-medium text-foreground">时间</span>
                <span>{formatDateTime(detail.date)}</span>
              </div>
            </div>

            <Separator />

            {detail.html_body || detail.text_body ? (
              <Tabs
                value={activeBodyTab}
                onValueChange={(v) => setActiveBodyTab(v as "html" | "text")}
                className="flex flex-1 flex-col overflow-hidden"
              >
                <TabsList className="shrink-0 self-start">
                  {detail.html_body && <TabsTrigger value="html">HTML</TabsTrigger>}
                  {detail.text_body && <TabsTrigger value="text">纯文本</TabsTrigger>}
                </TabsList>
                {detail.html_body && (
                  <TabsContent value="html" className="mt-2 flex-1 overflow-hidden">
                    <iframe
                      srcDoc={detail.html_body}
                      sandbox="allow-same-origin"
                      className="h-full w-full rounded-md border"
                      style={{ border: "none" }}
                      title="邮件内容"
                    />
                  </TabsContent>
                )}
                {detail.text_body && (
                  <TabsContent value="text" className="mt-2 flex-1 overflow-hidden">
                    <ScrollArea className="h-full rounded-md border p-4">
                      <pre className="whitespace-pre-wrap break-words font-sans text-sm">{detail.text_body}</pre>
                    </ScrollArea>
                  </TabsContent>
                )}
              </Tabs>
            ) : (
              <div className="flex flex-1 items-center justify-center text-sm text-muted-foreground">
                此邮件没有正文内容
              </div>
            )}
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

export function EmailAccountDetails({ accountId, open, onOpenChange }: EmailAccountDetailsProps) {
  const [account, setAccount] = useState<EmailAccount | null>(null);
  const [accountLoading, setAccountLoading] = useState(false);
  const [messages, setMessages] = useState<EmailMessageSummary[]>([]);
  const [messagesTotal, setMessagesTotal] = useState(0);
  const [messagesOffset, setMessagesOffset] = useState(0);
  const [mailbox, setMailbox] = useState("INBOX");
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<EmailMessageDetail | null>(null);

  useEffect(() => {
    if (!open || !accountId) {
      return;
    }

    setAccount(null);
    setMessages([]);
    setMessagesTotal(0);
    setMessagesOffset(0);
    setMailbox("INBOX");

    const loadAccount = async () => {
      try {
        setAccountLoading(true);
        const next = await api.getEmailAccount(accountId);
        setAccount(next);
        if (next.graph_summary?.mailbox) {
          setMailbox(next.graph_summary.mailbox);
        }
      } catch (error) {
        toast.error("加载邮箱配置失败: " + extractErrorMessage(error));
      } finally {
        setAccountLoading(false);
      }
    };

    void loadAccount();
  }, [open, accountId]);

  useEffect(() => {
    if (!open || !accountId) {
      return;
    }
    void loadMessages(messagesOffset);
  }, [open, accountId, mailbox, messagesOffset]);

  const loadMessages = async (offset: number) => {
    if (!accountId) {
      return;
    }
    try {
      setLoading(true);
      const res = await api.listEmailMessages(accountId, {
        limit: MESSAGE_PAGE_SIZE,
        offset,
        mailbox,
      });
      setMessages(res.items ?? []);
      setMessagesTotal(res.total ?? 0);
      if (res.mailbox) {
        setMailbox(res.mailbox);
      }
    } catch (error) {
      toast.error("加载邮件列表失败: " + extractErrorMessage(error));
    } finally {
      setLoading(false);
    }
  };

  const openMessage = async (messageId: string) => {
    if (!accountId) {
      return;
    }
    setDetail(null);
    setDetailOpen(true);
    setDetailLoading(true);
    try {
      const res = await api.getEmailMessage(accountId, messageId, mailbox);
      setDetail(res);
    } catch (error) {
      toast.error("加载邮件失败: " + extractErrorMessage(error));
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  const openLatest = async () => {
    if (!accountId) {
      return;
    }
    setDetail(null);
    setDetailOpen(true);
    setDetailLoading(true);
    try {
      const res = await api.getLatestEmailMessage(accountId, mailbox);
      if (!res.found || !res.item) {
        toast.info("当前邮箱夹暂无邮件");
        setDetailOpen(false);
        return;
      }
      setDetail(res.item);
    } catch (error) {
      toast.error("加载最新邮件失败: " + extractErrorMessage(error));
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent side="right" className="flex w-[900px] max-w-[900px] flex-col p-0 sm:max-w-[900px]">
          <SheetHeader className="border-b p-6">
            <SheetTitle>{account?.address ?? `邮箱 ${accountId ?? ""}`}</SheetTitle>
            <SheetDescription>查看邮箱配置摘要和收件箱内容。</SheetDescription>
          </SheetHeader>

          <div className="flex flex-1 flex-col gap-4 overflow-hidden p-6">
            {accountLoading && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                加载邮箱配置中...
              </div>
            )}

            <div className="shrink-0 flex items-center justify-between">
              <Button variant="outline" size="sm" onClick={openLatest} disabled={detailLoading}>
                {detailLoading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Mail className="mr-2 h-4 w-4" />}
                查看最新邮件
              </Button>
              <Button variant="outline" size="sm" onClick={() => void loadMessages(messagesOffset)} disabled={loading}>
                {loading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <RefreshCw className="mr-2 h-4 w-4" />}
                刷新
              </Button>
            </div>

            <div className="text-sm text-muted-foreground">
              当前邮箱夹: <span className="font-mono text-foreground">{mailbox}</span>
              {" · "}
              共 {messagesTotal} 封，当前从第 {messagesOffset + 1} 封开始
            </div>

            <div className="flex flex-1 flex-col overflow-hidden rounded-md border">
              <ScrollArea className="flex-1">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>ID</TableHead>
                      <TableHead>发件人</TableHead>
                      <TableHead>主题</TableHead>
                      <TableHead>时间</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {messages.map((msg) => (
                      <TableRow key={msg.id} className="cursor-pointer hover:bg-muted/50" onClick={() => void openMessage(msg.id)}>
                        <TableCell className="max-w-[180px] truncate font-mono text-xs" title={msg.id}>
                          {msg.id}
                        </TableCell>
                        <TableCell className="max-w-[220px] truncate" title={msg.from}>
                          {msg.from}
                        </TableCell>
                        <TableCell className="max-w-[320px] truncate" title={msg.subject}>
                          {msg.subject || "(无主题)"}
                        </TableCell>
                        <TableCell>{formatDateTime(msg.date)}</TableCell>
                      </TableRow>
                    ))}
                    {!loading && messages.length === 0 && (
                      <TableRow>
                        <TableCell colSpan={4} className="h-24 text-center">
                          <div className="flex flex-col items-center gap-2 text-muted-foreground">
                            <Mail className="h-8 w-8 opacity-50" />
                            <p>收件箱暂无邮件</p>
                          </div>
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </ScrollArea>
              <div className="border-t p-3">
                <Pagination total={messagesTotal} limit={MESSAGE_PAGE_SIZE} offset={messagesOffset} onPageChange={setMessagesOffset} />
              </div>
            </div>
          </div>
        </SheetContent>
      </Sheet>

      <MessageDetailDialog detail={detail} loading={detailLoading} open={detailOpen} onOpenChange={setDetailOpen} />
    </>
  );
}
