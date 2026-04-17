import type { SidebarsConfig } from "@docusaurus/plugin-content-docs";

const sidebars: SidebarsConfig = {
  docsSidebar: [
    {
      type: "category",
      label: "Getting Started",
      items: [
        "getting-started/install",
        "getting-started/quickstart"
      ]
    },
    "commands",
    "rules",
    "configuration",
    "architecture",
    "contributing",
    "roadmap",
    "releases",
    "faq"
  ]
};

export default sidebars;
