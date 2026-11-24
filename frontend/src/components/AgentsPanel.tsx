import { useEffect, useState } from "react";
import { Clipboard } from "@wailsio/runtime";

import type { Session } from "../../bindings/doter/internal/agents/models";
import { useAgentSessions } from "../hooks/useAgentSessions";
import { formatRelTime, formatTokens } from "../lib/format";
import { resumeCommand } from "../lib/resume";
import { AgentIcon } from "./AgentIcon";

const AGENT_LABELS: Record<string, string> = {
  claude: "Claude Code",
  codex: "Codex",
  opencode: "OpenCode",
  pi: "Pi",
  omp: "Oh My Pi",
};

const STATUS_LABELS: Record<string, string> = {
  running: "运行中",
  // waiting means the turn is blocked on a tool call the user has to approve
  // or answer; a finished turn is done even while its CLI stays open.
  waiting: "等待输入",
  done: "已完成",
};

const strokeProps = {
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.8,
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
  "aria-hidden": true,
};

/** The agent session monitor tab. */
export function AgentsPanel({ onFocusIsland }: { onFocusIsland: () => void }) {
  const { snapshot, live } = useAgentSessions();
  const sessions = snapshot?.sessions ?? [];

  // Relative timestamps and runtimes stay fresh while the panel is visible.
  const [, setTick] = useState(0);
  useEffect(() => {
    const timer = window.setInterval(() => setTick((value) => value + 1), 10_000);
    return () => window.clearInterval(timer);
  }, []);

  const totalTokens = (snapshot?.inputTokens ?? 0) + (snapshot?.outputTokens ?? 0);

  return (
    <div className="agents-panel">
      <div className="agents-summary">
        <div className="agents-stat is-running">
          <b>{snapshot?.running ?? 0}</b>
          <span>运行中</span>
        </div>
        <div className="agents-stat is-waiting">
          <b>{snapshot?.waiting ?? 0}</b>
          <span>等待输入</span>
        </div>
        <div className="agents-stat">
          <b>{formatTokens(totalTokens)}</b>
          <span>Token 合计</span>
        </div>
        <div className="agents-live" data-live={live}>
          <span className="brand-dot is-live" />
          实时
        </div>
      </div>

      <div className="agents-list">
        {sessions.length === 0 ? (
          <div className="agents-empty">
            <svg viewBox="0 0 24 24" {...strokeProps}>
              <rect x="3.5" y="4.5" width="17" height="15" rx="3.5" />
              <path d="M7.5 9 10 11.5 7.5 14M12.5 15h4" />
            </svg>
            <b>没有检测到 Agent 会话</b>
            <span>运行 claude、codex、opencode 或 pi 后，这里会显示实时状态</span>
          </div>
        ) : (
          sessions.map((session) => (
            <AgentRow
              key={`${session.agent}:${session.id}`}
              session={session}
              onFocusIsland={onFocusIsland}
            />
          ))
        )}
      </div>
    </div>
  );
}

function AgentRow({ session, onFocusIsland }: { session: Session; onFocusIsland: () => void }) {
  const title = session.title || session.projectName || session.id.slice(0, 8);
  const status = STATUS_LABELS[session.status] ?? session.status;
  const [receipt, setReceipt] = useState<"copied" | "failed" | null>(null);

  useEffect(() => {
    if (!receipt) return;
    const timer = window.setTimeout(() => setReceipt(null), 1_500);
    return () => window.clearTimeout(timer);
  }, [receipt]);

  // The whole row copies: the project name is the row's identity, and a
  // one-target row is easier to hit than a small inline link.
  const copy = () => {
    const command = resumeCommand(session);
    if (!command) {
      setReceipt("failed");
      return;
    }
    // The runtime's clipboard call goes straight to the native window, so it
    // works without a matching service method.
    Clipboard.SetText(command).then(
      () => setReceipt("copied"),
      () => setReceipt("failed"),
    );
  };

  return (
    <article
      className="agent-row"
      data-agent={session.agent}
      data-status={session.status}
      role="button"
      tabIndex={0}
      aria-label={`复制 ${title} 的恢复命令`}
      title={session.project || title}
      // A non-activating panel drops the first click; activating on hover
      // (as the search inputs do) makes the following click land.
      onPointerEnter={onFocusIsland}
      onPointerDown={(event) => {
        if (event.button !== 0) return;
        onFocusIsland();
        // Copy on pointerdown, like the pin button: the panel's first click
        // is consumed while it is not yet key, so waiting for `click` loses it.
        copy();
      }}
      onKeyDown={(event) => {
        if (event.key !== "Enter" && event.key !== " ") return;
        event.preventDefault();
        copy();
      }}
    >
      <span className="agent-badge" aria-hidden="true">
        <AgentIcon agent={session.agent} className="agent-icon" />
      </span>

      <div className="agent-main">
        <div className="agent-headline">
          <span className="agent-title" title={title}>
            {title}
          </span>
        </div>

        <div className="agent-meta">
          <span className="agent-name">{AGENT_LABELS[session.agent] ?? session.agent}</span>
          <span className="agent-sep">·</span>
          <span
            className="agent-project"
            data-receipt={receipt || undefined}
          >
            {receipt === "copied" ? "已复制" : receipt === "failed" ? "复制失败" : session.projectName || "—"}
          </span>
          {session.branch && (
            <>
              <span className="agent-sep">·</span>
              <span className="agent-branch">{session.branch}</span>
            </>
          )}
          {session.model && (
            <>
              <span className="agent-sep">·</span>
              <span className="agent-model" title={session.model}>
                {session.model}
              </span>
            </>
          )}
        </div>

        <div className="agent-foot">
          <span>↑ {formatTokens(session.inputTokens)}</span>
          <span>↓ {formatTokens(session.outputTokens)}</span>
          {session.cachedTokens > 0 && <span>缓存 {formatTokens(session.cachedTokens)}</span>}
          {session.contextTokens > 0 && <span>ctx {formatTokens(session.contextTokens)}</span>}
          {session.turns > 0 && <span>{session.turns} 轮</span>}
        </div>
      </div>

      <div className="agent-side">
        <span className={`agent-status is-${session.status}`}>
          <i aria-hidden="true" />
          {status}
        </span>
        <span className="agent-time">{formatRelTime(session.lastActivity)}</span>
      </div>
    </article>
  );
}

