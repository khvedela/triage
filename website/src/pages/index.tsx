import Layout from "@theme/Layout";
import Link from "@docusaurus/Link";
import ComparisonTable from "../components/ComparisonTable";
import InstallTabs from "../components/InstallTabs";
import ScenarioCard from "../components/ScenarioCard";
import SignalChip from "../components/SignalChip";
import TerminalFrame from "../components/TerminalFrame";
import ArchitectureFlow from "../components/docs/ArchitectureFlow";
import styles from "./index.module.css";

export default function Home(): JSX.Element {
  return (
    <Layout
      title="triage | Kubernetes diagnosis with ranked findings"
      description="triage is a kubectl-native diagnostic CLI for broken Kubernetes workloads."
    >
      <main className={styles.page}>
        <section className={styles.hero}>
          <div className="container">
            <div className={styles.heroGrid}>
              <div>
                <p className={styles.eyebrow}>Kubernetes diagnostics without the archaeology</p>
                <h1 className={styles.heroTitle}>Turn broken workload symptoms into ranked root causes.</h1>
                <p className={styles.heroBody}>
                  triage is a kubectl-native CLI that cross-references pod status, events, owner refs, services,
                  endpoints, PVCs, and RBAC in one pass. It tells responders what is broken, why, and what command to
                  run next.
                </p>
                <div className={styles.heroActions}>
                  <Link className="button button--primary button--lg" to="/docs/getting-started/install">
                    Install triage
                  </Link>
                  <Link className="button button--secondary button--lg" to="/sandbox?scenario=crashloop">
                    Open sandbox
                  </Link>
                </div>
                <div className={styles.heroSummary}>
                  <span>23 built-in rules</span>
                  <span>Text, JSON, Markdown output</span>
                  <span>Read-only by design</span>
                </div>
              </div>
              <div className={styles.heroRail}>
                <TerminalFrame label="incident:default/crashloop-demo">
                  {`ⓧ CRITICAL  [high confidence]  TRG-POD-CRASHLOOPBACKOFF
Container "app" is in CrashLoopBackOff (5 restarts)

Evidence:
  • pod.status.containerStatuses[app].state.waiting.reason = CrashLoopBackOff
  • pod.status.containerStatuses[app].restartCount = 5

Next commands:
  $ kubectl logs -n default crashloop-demo -c app --previous
  $ kubectl describe pod -n default crashloop-demo`}
                </TerminalFrame>
                <p className={styles.heroNote}>
                  Built for responders who need the likely root cause, the evidence behind it, and the shortest path
                  to confirmation.
                </p>
              </div>
            </div>
          </div>
        </section>

        <section className={styles.section}>
          <div className="container">
            <div className={styles.sectionHeader}>
              <p className={styles.eyebrow}>Why triage</p>
              <h2>Not another wrapper around `kubectl describe`.</h2>
              <p>
                triage keeps the raw cluster facts, but adds ranking, rule IDs, evidence correlation, and immediate
                follow-up commands so responders can move from symptom to fix faster.
              </p>
            </div>
            <ComparisonTable />
          </div>
        </section>

        <section className={styles.section}>
          <div className="container">
            <div className={styles.sectionHeader}>
              <p className={styles.eyebrow}>Core benefits</p>
              <h2>Built for operators who need the signal first.</h2>
            </div>
            <div className={styles.cardGrid}>
              <article className={styles.benefitCard}>
                <SignalChip tone="critical" label="rank first" />
                <h3>Ranked diagnosis</h3>
                <p>Severity and confidence push the incident-driving finding to the top instead of leaving clues scattered across commands.</p>
              </article>
              <article className={styles.benefitCard}>
                <SignalChip tone="medium" label="evidence linked" />
                <h3>Evidence preserved</h3>
                <p>Every finding carries contract-grade evidence so automation and humans can reason from the same facts.</p>
              </article>
              <article className={styles.benefitCard}>
                <SignalChip tone="info" label="actionable next step" />
                <h3>Exact next command</h3>
                <p>Findings include pasteable follow-up commands, which shortens the loop from detection to remediation.</p>
              </article>
            </div>
          </div>
        </section>

        <section className={styles.section}>
          <div className="container">
            <div className={styles.sectionHeader}>
              <p className={styles.eyebrow}>Scenario-led demos</p>
              <h2>Explore the failures triage is meant to cut through.</h2>
            </div>
            <div className={styles.cardGrid}>
              <ScenarioCard
                title="CrashLoopBackOff"
                summary="See how triage prioritizes runtime failure, retains evidence, and keeps follow-up commands copyable."
                ruleId="TRG-POD-CRASHLOOPBACKOFF"
                severity="critical"
                href="/sandbox?scenario=crashloop"
              />
              <ScenarioCard
                title="Missing ConfigMap"
                summary="Watch a broken dependency turn into a concrete configuration diagnosis with object-level evidence."
                ruleId="TRG-POD-MISSING-CONFIGMAP"
                severity="high"
                href="/sandbox?scenario=missing-configmap"
              />
              <ScenarioCard
                title="Stuck rollout"
                summary="Inspect how triage connects controller-level rollout failure to the underlying image pull problem."
                ruleId="TRG-DEPLOY-ROLLOUT-STUCK"
                severity="critical"
                href="/sandbox?scenario=stuck-rollout"
              />
            </div>
          </div>
        </section>

        <section className={styles.section}>
          <div className="container">
            <div className={styles.sectionHeader}>
              <p className={styles.eyebrow}>Architecture</p>
              <h2>One-shot CLI, request-scoped cache, static rule engine.</h2>
              <p>
                The product stays fast because it does one diagnosis pass, builds a request-scoped cache, and runs
                compiled rules over explicit Kubernetes signals instead of trying to become a long-lived control
                plane.
              </p>
            </div>
            <ArchitectureFlow />
          </div>
        </section>

        <section className={styles.section}>
          <div className="container">
            <div className={styles.installBand}>
              <div className={styles.sectionHeader}>
                <p className={styles.eyebrow}>Install</p>
                <h2>Use the release flow that matches your environment.</h2>
                <p>
                  Releases ship as signed binaries with SBOMs. Source installs use the public module path, and kubectl
                  plugin mode is built into the same binary.
                </p>
              </div>
              <InstallTabs />
            </div>
          </div>
        </section>

        <section className={styles.section}>
          <div className="container">
            <div className={styles.sectionHeader}>
              <p className={styles.eyebrow}>Release trust</p>
              <h2>Distribution details that matter in production environments.</h2>
            </div>
            <div className={styles.trustGrid}>
              <article className={styles.trustCard}>
                <h3>Signed artifacts</h3>
                <p>GoReleaser publishes checksums and cosign signatures so teams can verify exactly what they install.</p>
              </article>
              <article className={styles.trustCard}>
                <h3>SBOMs included</h3>
                <p>Each archive ships with SBOM material so provenance and dependency review do not require a second pipeline.</p>
              </article>
              <article className={styles.trustCard}>
                <h3>GitHub-native releases</h3>
                <p>Binary downloads, release notes, and future package integrations all anchor on the public GitHub Releases feed.</p>
              </article>
            </div>
          </div>
        </section>
      </main>
    </Layout>
  );
}
