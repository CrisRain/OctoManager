import { useEffect, useState } from "react";
import { CheckCircle2, CircleX, Loader2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  OUTLOOK_OAUTH_CALLBACK_MESSAGE,
  type OutlookOAuthCallbackMessage,
} from "@/pages/email/components/outlook-oauth-bridge";

type CallbackStatus = "processing" | "success" | "error" | "detached";

export function OAuthCallbackPage() {
  const [status, setStatus] = useState<CallbackStatus>("processing");
  const [message, setMessage] = useState("正在处理 OAuth 回调...");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const payload: OutlookOAuthCallbackMessage = {
      type: OUTLOOK_OAUTH_CALLBACK_MESSAGE,
      code: params.get("code") ?? undefined,
      state: params.get("state") ?? undefined,
      error: params.get("error") ?? undefined,
      error_description: params.get("error_description") ?? undefined,
    };

    if (window.opener && !window.opener.closed) {
      window.opener.postMessage(payload, window.location.origin);

      if (payload.error) {
        setStatus("error");
        setMessage(payload.error_description || payload.error);
      } else if (payload.code) {
        setStatus("success");
        setMessage("授权完成，可返回原始窗口。");
      } else {
        setStatus("error");
        setMessage("回调未包含授权码。");
      }

      const closeTimer = window.setTimeout(() => window.close(), 300);
      return () => window.clearTimeout(closeTimer);
    }

    if (payload.error) {
      setStatus("error");
      setMessage(payload.error_description || payload.error);
      return;
    }

    if (payload.code) {
      setStatus("detached");
      setMessage("授权成功，但未检测到父窗口，可关闭此页面。");
      return;
    }

    setStatus("error");
    setMessage("回调参数缺失。");
  }, []);

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <Card className="w-full max-w-xl">
        <CardHeader>
          <CardTitle>OAuth 回调</CardTitle>
          <CardDescription>
            将 Outlook OAuth 结果转发回管理页面。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center gap-2 text-sm">
            {status === "processing" ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            {status === "success" ? <CheckCircle2 className="h-4 w-4 text-green-600" /> : null}
            {status === "detached" ? <CheckCircle2 className="h-4 w-4 text-amber-600" /> : null}
            {status === "error" ? <CircleX className="h-4 w-4 text-destructive" /> : null}
            <span>{message}</span>
          </div>
          <p className="text-xs text-muted-foreground">
            当前 URL：<code className="font-mono">{window.location.pathname}</code>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
