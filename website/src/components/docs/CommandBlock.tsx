import styles from "./docs.module.css";

type Props = {
  eyebrow?: string;
  title: string;
  description: string;
  command: string | string[];
  caption?: string;
};

export default function CommandBlock({ eyebrow, title, description, command, caption }: Props): JSX.Element {
  const lines = Array.isArray(command) ? command.join("\n") : command;

  return (
    <section className={styles.commandBlock}>
      <div className={styles.commandHeader}>
        {eyebrow ? <p className={styles.eyebrow}>{eyebrow}</p> : null}
        <h3>{title}</h3>
        <p>{description}</p>
      </div>
      <div className={styles.shell}>
        <pre>{lines}</pre>
      </div>
      {caption ? <p className={styles.caption}>{caption}</p> : null}
    </section>
  );
}
