import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";
import { themes as prismThemes } from "prism-react-renderer";

const config: Config = {
  title: "kubediag",
  tagline: "Ranked Kubernetes diagnosis with evidence, confidence, and the next command to run.",
  favicon: "img/favicon.svg",
  url: "https://khvedela.github.io",
  baseUrl: "/triage/",
  organizationName: "khvedela",
  projectName: "kubediag",
  trailingSlash: false,
  onBrokenLinks: "throw",
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: "throw"
    }
  },
  i18n: {
    defaultLocale: "en",
    locales: ["en"]
  },
  themes: [],
  presets: [
    [
      "classic",
      {
        docs: {
          routeBasePath: "docs",
          sidebarPath: "./sidebars.ts",
          lastVersion: "v0.2.0",
          versions: {
            current: {
              label: "Next (unreleased)",
              path: "next",
              banner: "unreleased"
            },
            "v0.2.0": {
              label: "v0.2.0",
              path: "/"
            },
            "v0.1.0": {
              label: "v0.1.0",
              path: "v0.1.0"
            }
          },
          includeCurrentVersion: true,
          editUrl: "https://github.com/khvedela/kubediag/tree/main/website/",
          showLastUpdateTime: false
        },
        blog: false,
        theme: {
          customCss: "./src/css/custom.css"
        }
      } satisfies Preset.Options
    ]
  ],
  themeConfig: {
    image: "img/logo.svg",
    navbar: {
      title: "kubediag",
      logo: {
        alt: "kubediag logo",
        src: "img/logo.svg"
      },
      items: [
        { to: "/sandbox", label: "Sandbox", position: "left" },
        {
          type: "docSidebar",
          sidebarId: "docsSidebar",
          label: "Docs",
          position: "left",
          activeBaseRegex: "^/docs/"
        },
        {
          type: "docsVersionDropdown",
          position: "right",
          dropdownItemsAfter: [
            {
              to: "https://github.com/khvedela/kubediag/releases",
              label: "GitHub releases"
            }
          ]
        },
        {
          href: "https://github.com/khvedela/kubediag",
          label: "GitHub",
          position: "right"
        }
      ]
    },
    footer: {
      links: [
        {
          title: "Product",
          items: [
            { label: "Sandbox", to: "/sandbox" },
            { label: "Install", to: "/docs/getting-started/install" },
            { label: "Releases", to: "/docs/releases" }
          ]
        },
        {
          title: "Documentation",
          items: [
            { label: "Quickstart", to: "/docs/getting-started/quickstart" },
            { label: "Commands", to: "/docs/commands" },
            { label: "Rules", to: "/docs/rules" }
          ]
        },
        {
          title: "Project",
          items: [
            { label: "Repository", href: "https://github.com/khvedela/kubediag" },
            { label: "Issues", href: "https://github.com/khvedela/kubediag/issues" },
            { label: "Security", href: "https://github.com/khvedela/kubediag/security" }
          ]
        }
      ],
      copyright: `Copyright © ${new Date().getFullYear()} kubediag contributors.`
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.vsDark
    },
    colorMode: {
      defaultMode: "light",
      disableSwitch: false,
      respectPrefersColorScheme: false
    }
  } satisfies Preset.ThemeConfig
};

export default config;
