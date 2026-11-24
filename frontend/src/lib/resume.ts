import type { Session } from "../../bindings/doter/internal/agents/models";

/** A shell command that reopens the session in its project directory. */
export function resumeCommand(session: Session): string {
  let resume: string;
  switch (session.agent) {
    case "claude":
      resume = `claude --resume ${quote(session.id)}`;
      break;
    case "codex":
      resume = `codex resume ${quote(session.id)}`;
      break;
    case "opencode":
      resume = `opencode --session ${quote(session.id)}`;
      break;
    case "pi":
      resume = `pi --session ${quote(session.id)}`;
      break;
    case "omp":
      resume = `omp --resume=${quote(session.id)}`;
      break;
    default:
      return "";
  }
  return session.project ? `cd ${quote(session.project)} && ${resume}` : resume;
}

/** Wraps values a shell would otherwise split or expand. */
function quote(value: string): string {
  if (value && !/[\s'"\\$`&|;<>()*?[\]{}!#~]/.test(value)) return value;
  return `'${value.split("'").join(`'\\''`)}'`;
}
