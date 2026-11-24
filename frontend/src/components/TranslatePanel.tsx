import { useRef, useState } from "react";
import type { KeyboardEvent } from "react";

import { Translate } from "../../bindings/doter/internal/services/translateservice";
import type { Result } from "../../bindings/doter/internal/translate/models";
import { Card } from "./Card";

/** Target languages offered in the header select. */
const TARGETS = [
  { id: "auto", label: "自动" },
  { id: "zh-CN", label: "中文" },
  { id: "en", label: "English" },
  { id: "ja", label: "日本語" },
  { id: "ko", label: "한국어" },
];

const LANG_LABELS: Record<string, string> = {
  "zh-CN": "中文",
  "zh-TW": "繁體中文",
  en: "English",
  ja: "日本語",
  ko: "한국어",
};

const CJK = /[㐀-鿿]/;
/** Long enough for a paragraph; keeps the GET URL inside the provider limit. */
const MAX_CHARS = 4000;

function langLabel(code: string): string {
  return LANG_LABELS[code] ?? code;
}

/** Auto mode: Chinese goes out to English, everything else comes back to Chinese. */
function effectiveTarget(text: string, target: string): string {
  if (target !== "auto") return target;
  return CJK.test(text) ? "en" : "zh-CN";
}

/** The translate tab: paste text, pick a target, get one result. */
export function TranslatePanel() {
  const [text, setText] = useState("");
  const [target, setTarget] = useState("auto");
  const [result, setResult] = useState<Result | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);
  // Only the newest request may write state; two quick presses can land out of
  // order otherwise.
  const seq = useRef(0);

  const change = (value: string) => {
    setText(value);
    setCopied(false);
  };

  const run = async () => {
    const value = text.trim();
    if (!value || busy) return;
    const id = ++seq.current;
    setBusy(true);
    setError("");
    setCopied(false);
    const res = await Translate(value, effectiveTarget(value, target));
    if (seq.current !== id) return;
    setBusy(false);
    if (!res) {
      setResult(null);
      setError("翻译失败，检查网络后重试");
      return;
    }
    setResult(res);
  };

  const copy = async () => {
    if (!result) return;
    try {
      await navigator.clipboard.writeText(result.text);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      // Clipboard permission denied: the text stays selectable.
    }
  };

  const onKey = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      void run();
    }
  };

  return (
    <Card
      label="Translate"
      accent="var(--accent-mem)"
      wide
      actions={
        <select
          className="tr-target"
          value={target}
          disabled={busy}
          aria-label="目标语言"
          onChange={(event) => setTarget(event.target.value)}
        >
          {TARGETS.map((item) => (
            <option key={item.id} value={item.id}>
              {item.label}
            </option>
          ))}
        </select>
      }
    >
      <div className="tr-panel">
        <textarea
          className="tr-input"
          value={text}
          maxLength={MAX_CHARS}
          spellCheck={false}
          placeholder="输入或粘贴文字…"
          onChange={(event) => change(event.target.value)}
          onKeyDown={onKey}
        />

        <div className="tr-run">
          <span className="tr-hint">
            {text.trim() ? `${text.trim().length} / ${MAX_CHARS} 字符` : "自动识别源语言 · ⌘↵ 翻译"}
          </span>
          <button
            type="button"
            className="tr-go"
            disabled={busy || !text.trim()}
            onClick={() => void run()}
          >
            {busy ? "翻译中…" : "翻译"}
          </button>
        </div>

        {error && <p className="tr-error">{error}</p>}

        {result && (
          <div className="tr-result">
            <p className="tr-text">{result.text}</p>
            <div className="tr-meta">
              <span>
                {result.source ? `${langLabel(result.source)} → ${langLabel(result.target)}` : langLabel(result.target)}
                {" · "}
                {result.provider}
              </span>
              <button type="button" className="tr-copy" onClick={() => void copy()}>
                {copied ? "已复制" : "复制"}
              </button>
            </div>
          </div>
        )}
      </div>
    </Card>
  );
}