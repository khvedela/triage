import styles from "./docs.module.css";

type Props = {
  eyebrow?: string;
  title: string;
  lede: string;
  meta?: string[];
  compact?: boolean;
};

export default function DocHero({ eyebrow, title, lede, meta = [], compact = false }: Props): JSX.Element {
  return (
    <section className={`${styles.hero} ${compact ? styles.heroCompact : ""}`}>
      {eyebrow ? <p className={styles.eyebrow}>{eyebrow}</p> : null}
      <h1>{title}</h1>
      <p className={styles.lede}>{lede}</p>
      {meta.length > 0 ? (
        <div className={styles.meta}>
          {meta.map((item) => (
            <span key={item} className={styles.metaItem}>
              {item}
            </span>
          ))}
        </div>
      ) : null}
    </section>
  );
}
