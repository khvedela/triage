import clsx from "clsx";
import styles from "./ui.module.css";

type Props = {
  tone: "critical" | "high" | "medium" | "info";
  label: string;
};

export default function SignalChip({ tone, label }: Props): JSX.Element {
  return <span className={clsx(styles.signalChip, styles[tone])}>{label}</span>;
}
