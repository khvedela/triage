import type { ReactNode } from "react";
import styles from "./ui.module.css";

type Props = {
  label: string;
  actions?: ReactNode;
  children: ReactNode;
};

export default function TerminalFrame({ label, actions, children }: Props): JSX.Element {
  return (
    <div className={styles.terminalFrame}>
      <div className={styles.terminalTop}>
        <div className={styles.terminalDots} aria-hidden="true">
          <span className={styles.terminalDot} style={{ background: "#ff8b80" }} />
          <span className={styles.terminalDot} style={{ background: "#f6c874" }} />
          <span className={styles.terminalDot} style={{ background: "#8cd3c2" }} />
        </div>
        <strong className={styles.terminalLabel}>{label}</strong>
        {actions ? <div className={styles.terminalActions}>{actions}</div> : null}
      </div>
      <div className={styles.terminalViewport}>
        <pre className={styles.terminalBody}>{children}</pre>
      </div>
    </div>
  );
}
