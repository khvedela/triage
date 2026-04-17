import styles from "./docs.module.css";

type Item = {
  label?: string;
  title: string;
  body: string;
};

type Props = {
  items: Item[];
  columns?: 2 | 3;
};

export default function InfoGrid({ items, columns = 3 }: Props): JSX.Element {
  return (
    <section className={`${styles.grid} ${columns === 2 ? styles.gridTwo : styles.gridThree}`}>
      {items.map((item) => (
        <article key={`${item.title}-${item.body}`} className={styles.card}>
          {item.label ? <span className={styles.cardLabel}>{item.label}</span> : null}
          <h3>{item.title}</h3>
          <p>{item.body}</p>
        </article>
      ))}
    </section>
  );
}
