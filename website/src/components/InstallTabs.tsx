import { useState } from "react";
import styles from "./ui.module.css";
import TerminalFrame from "./TerminalFrame";

const tabs = [
  {
    id: "binary",
    label: "Binary",
    description: "Pull a signed release asset from GitHub Releases.",
    command: `curl -L https://github.com/khvedela/triage/releases/latest/download/triage_darwin_arm64.tar.gz | tar xz
chmod +x triage
mv triage /usr/local/bin/triage`
  },
  {
    id: "source",
    label: "Go install",
    description: "Install directly from source using the public module path.",
    command: "go install github.com/khvedela/triage@latest"
  },
  {
    id: "plugin",
    label: "kubectl plugin",
    description: "Build once, symlink as kubectl-triage, and run through kubectl.",
    command: `make build-plugin
export PATH="$(pwd)/bin:$PATH"
kubectl triage pod my-pod -n default`
  }
];

export default function InstallTabs(): JSX.Element {
  const [active, setActive] = useState(tabs[0]);

  return (
    <div className={styles.installTabs}>
      <div className={styles.installTabButtons}>
        {tabs.map((tab) => (
          <button
            key={tab.id}
            className={`${styles.installTabButton} ${active.id === tab.id ? styles.installTabButtonActive : ""}`}
            type="button"
            onClick={() => setActive(tab)}
          >
            {tab.label}
          </button>
        ))}
      </div>
      <p className="triage-muted triage-no-margin">{active.description}</p>
      <TerminalFrame label={`install:${active.id}`}>{active.command}</TerminalFrame>
    </div>
  );
}
