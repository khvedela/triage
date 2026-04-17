import styles from "./docs.module.css";

const steps = [
  {
    title: "CLI surface",
    body: "Cobra commands map user intent to one diagnosis target and a renderer choice."
  },
  {
    title: "Collector + cache",
    body: "Request-scoped cache warms likely resources and removes duplicate Kubernetes API calls."
  },
  {
    title: "Rule engine",
    body: "Compiled rules inspect explicit workload, event, network, storage, and RBAC signals."
  },
  {
    title: "Ranking",
    body: "Severity and confidence decide what responders should read first."
  },
  {
    title: "Renderers",
    body: "Text, JSON, and Markdown keep the same finding model for humans and automation."
  }
];

export default function ArchitectureFlow(): JSX.Element {
  return (
    <section aria-label="Architecture flow" className={styles.flow}>
      <div className={styles.flowRail}>
        {steps.map((step, index) => (
          <article key={step.title} className={styles.flowStep}>
            <div className={styles.flowMarker}>
              <span className={styles.flowIndex}>{index + 1}</span>
              {index < steps.length - 1 ? <span aria-hidden="true" className={styles.flowConnector} /> : null}
            </div>
            <div className={styles.flowContent}>
              <h3>{step.title}</h3>
              <p>{step.body}</p>
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}
