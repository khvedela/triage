import styles from "./ui.module.css";

export default function ComparisonTable(): JSX.Element {
  return (
    <div className={styles.comparisonTable}>
      <section className={styles.comparisonColumn}>
        <h3 className={styles.comparisonHeading}>Manual cluster archaeology</h3>
        <p className={styles.comparisonCopy}>
          Useful for raw detail, but the operator still has to correlate symptoms, owners, events, services, and next
          steps by hand.
        </p>
        <ul className={styles.comparisonList}>
          <li className={styles.comparisonItem}>
            <strong>Raw status first</strong>
            <span>Good at showing what Kubernetes knows, not why one signal matters more than another.</span>
          </li>
          <li className={styles.comparisonItem}>
            <strong>Cross-resource reasoning is manual</strong>
            <span>Events, services, endpoints, PVCs, and RBAC checks still have to be stitched together mentally.</span>
          </li>
          <li className={styles.comparisonItem}>
            <strong>No built-in next step</strong>
            <span>The operator decides which command to run next based on experience and time pressure.</span>
          </li>
        </ul>
      </section>

      <section className={styles.comparisonColumn}>
        <h3 className={styles.comparisonHeading}>triage workflow</h3>
        <p className={styles.comparisonCopy}>
          Designed for incident response: rank the likely cause, preserve the evidence, and make the confirmation path
          obvious.
        </p>
        <ul className={styles.comparisonList}>
          <li className={styles.comparisonItem}>
            <strong>Ranked findings</strong>
            <span>Severity and confidence push the likely incident driver to the top instead of treating all signals equally.</span>
          </li>
          <li className={styles.comparisonItem}>
            <strong>Evidence linked</strong>
            <span>Services, endpoints, events, rollout conditions, storage references, and RBAC gaps stay attached to the finding.</span>
          </li>
          <li className={styles.comparisonItem}>
            <strong>Action-oriented output</strong>
            <span>Each finding includes the next command to run, plus JSON and markdown output for automation and reports.</span>
          </li>
        </ul>
      </section>
    </div>
  );
}
