import clsx from "clsx";
import { useEffect, useState } from "react";
import RuleBadge from "./RuleBadge";
import SignalChip from "./SignalChip";
import TerminalFrame from "./TerminalFrame";
import { scenarios } from "../data/scenarios";
import styles from "./SandboxLab.module.css";

type OutputTab = "text" | "markdown" | "json";

function readScenarioFromUrl(): string {
  if (typeof window === "undefined") {
    return scenarios[0].slug;
  }

  const params = new URLSearchParams(window.location.search);
  return params.get("scenario") || scenarios[0].slug;
}

export default function SandboxLab(): JSX.Element {
  const [activeSlug, setActiveSlug] = useState(scenarios[0].slug);
  const [activeTab, setActiveTab] = useState<OutputTab>("text");
  const [yaml, setYaml] = useState(scenarios[0].yaml);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    const slug = readScenarioFromUrl();
    const nextScenario = scenarios.find((scenario) => scenario.slug === slug) || scenarios[0];
    setActiveSlug(nextScenario.slug);
    setYaml(nextScenario.yaml);
  }, []);

  const scenario = scenarios.find((item) => item.slug === activeSlug) || scenarios[0];
  const yamlChanged = yaml !== scenario.yaml;

  useEffect(() => {
    setYaml(scenario.yaml);
    setActiveTab("text");
    setCopied(false);
  }, [scenario.slug]);

  async function copyCommands(): Promise<void> {
    try {
      await navigator.clipboard.writeText(scenario.nextCommands.join("\n"));
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      setCopied(false);
    }
  }

  function switchScenario(slug: string): void {
    const next = scenarios.find((item) => item.slug === slug) || scenarios[0];
    setActiveSlug(next.slug);
    if (typeof window !== "undefined") {
      const url = new URL(window.location.href);
      url.searchParams.set("scenario", next.slug);
      window.history.replaceState({}, "", url.toString());
    }
  }

  return (
    <div className={styles.shell}>
      <div className={styles.header}>
        <div>
          <p className={styles.eyebrow}>Scenario Lab</p>
          <h1>Interactive diagnosis demos grounded in real Kubernetes failure fixtures.</h1>
        </div>
        <p className={styles.scenarioSummary}>
          Edit the manifest locally, inspect the cluster signals, and see how triage presents the incident. The
          sandbox stays deterministic by design: outputs remain curated to the selected scenario.
        </p>
      </div>

      <div className={styles.scenarioRail}>
        {scenarios.map((item) => (
          <button
            key={item.slug}
            type="button"
            className={clsx(styles.scenarioButton, item.slug === scenario.slug && styles.scenarioButtonActive)}
            onClick={() => switchScenario(item.slug)}
          >
            <div className={styles.scenarioTopRow}>
              <SignalChip tone={item.severity} label={item.severity} />
              <RuleBadge ruleId={item.primaryRule} />
            </div>
            <strong>{item.name}</strong>
            <span className={styles.scenarioSummary}>{item.summary}</span>
          </button>
        ))}
      </div>

      <div className={styles.grid}>
        <div className={styles.column}>
          <section className={styles.panel}>
            <p className={styles.eyebrow}>Manifest input</p>
            <h2>{scenario.headline}</h2>
            <p className={styles.scenarioSummary}>{scenario.summary}</p>
            <textarea
              className={styles.yamlArea}
              value={yaml}
              onChange={(event) => setYaml(event.target.value)}
              spellCheck={false}
              aria-label={`${scenario.name} manifest`}
            />
            <p className={styles.statusNote}>
              {yamlChanged
                ? "Local edits are visible immediately, but diagnosis output stays pinned to the selected scenario."
                : "This panel starts from a repository-backed fixture and can be edited safely in-browser."}
            </p>
          </section>

          <section className={styles.panel}>
            <p className={styles.eyebrow}>Cluster signals</p>
            <h3>What triage would anchor on first</h3>
            <ul className={styles.signalList}>
              {scenario.signals.map((signal) => (
                <li key={signal}>{signal}</li>
              ))}
            </ul>
          </section>
        </div>

        <div className={styles.column}>
          <section className={styles.panel}>
            <div className={styles.panelHeader}>
              <div>
                <p className={styles.eyebrow}>Diagnosis output</p>
                <h3>{scenario.primaryRule}</h3>
              </div>
              <div className={styles.tabs}>
                {(["text", "markdown", "json"] as OutputTab[]).map((tab) => (
                  <button
                    key={tab}
                    type="button"
                    className={clsx(styles.tabButton, activeTab === tab && styles.tabButtonActive)}
                    onClick={() => setActiveTab(tab)}
                  >
                    {tab}
                  </button>
                ))}
              </div>
            </div>
            <TerminalFrame label={`triage-output:${activeTab}`}>
              {activeTab === "text" ? scenario.outputs.text : activeTab === "markdown" ? scenario.outputs.markdown : scenario.outputs.json}
            </TerminalFrame>
          </section>

          <section className={styles.panel}>
            <p className={styles.eyebrow}>Why this finding wins</p>
            <ul className={styles.whyList}>
              {scenario.whyItWins.map((reason) => (
                <li key={reason}>{reason}</li>
              ))}
            </ul>
          </section>

          <section className={styles.panel}>
            <div className={styles.panelHeader}>
              <div>
                <p className={styles.eyebrow}>Next commands</p>
                <h3>Pasteable follow-up for responders</h3>
              </div>
              <button className={styles.copyButton} type="button" onClick={() => void copyCommands()}>
                {copied ? "Copied" : "Copy commands"}
              </button>
            </div>
            <ul className={styles.commandList}>
              {scenario.nextCommands.map((command) => (
                <li key={command}>
                  <code>{command}</code>
                </li>
              ))}
            </ul>
          </section>
        </div>
      </div>
    </div>
  );
}
