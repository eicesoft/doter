import type { CSSProperties, ReactNode } from "react";

interface CardProps {
  label: string;
  hint?: string;
  /** Interactive controls rendered at the card's top-right corner. */
  actions?: ReactNode;
  /** Fixed-width control placed directly behind the label, in the title row. */
  inline?: ReactNode;
  accent: string;
  wide?: boolean;
  /** Settings can switch a card off without unmounting its siblings. */
  hidden?: boolean;
  children: ReactNode;
}

/** A dashboard tile: title row, optional header controls, then the body. */
export function Card({ label, hint, actions, inline, accent, wide, hidden, children }: CardProps) {
  if (hidden) return null;
  return (
    <section
      className={wide ? "card card-wide" : "card"}
      style={{ "--accent": accent } as CSSProperties}
    >
      <div className="card-head">
        <span className="card-label">{label}</span>
        {inline}
        {actions ? (
          <div className="card-actions">{actions}</div>
        ) : (
          hint && <span className="card-hint">{hint}</span>
        )}
      </div>
      <div className="card-body">{children}</div>
    </section>
  );
}
