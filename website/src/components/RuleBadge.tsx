import styles from "./ui.module.css";

type Props = {
  ruleId: string;
};

export default function RuleBadge({ ruleId }: Props): JSX.Element {
  return <span className={styles.ruleBadge}>{ruleId}</span>;
}
