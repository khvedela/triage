import styles from "./docs.module.css";

const categories = [
  "Access",
  "Configuration",
  "Image",
  "Networking",
  "Probes",
  "ResourcePressure",
  "Rollout",
  "Runtime",
  "Scheduling",
  "Storage"
];

export default function RulesCategoryNav(): JSX.Element {
  return (
    <nav className={styles.categoryNav} aria-label="Rule categories">
      {categories.map((category) => (
        <a key={category} className={styles.categoryLink} href={`#${category.toLowerCase()}`}>
          {category}
        </a>
      ))}
    </nav>
  );
}
