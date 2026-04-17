import Link from "@docusaurus/Link";
import RuleBadge from "./RuleBadge";
import SignalChip from "./SignalChip";
import styles from "./ui.module.css";

type Props = {
  title: string;
  summary: string;
  ruleId: string;
  severity: "critical" | "high" | "medium" | "info";
  href: string;
};

export default function ScenarioCard({
  title,
  summary,
  ruleId,
  severity,
  href
}: Props): JSX.Element {
  return (
    <article className={styles.scenarioCard}>
      <div className="triage-inline-row">
        <SignalChip tone={severity} label={severity} />
        <RuleBadge ruleId={ruleId} />
      </div>
      <h3>{title}</h3>
      <p>{summary}</p>
      <Link className="button button--outline button--secondary" to={href}>
        Open scenario
      </Link>
    </article>
  );
}
