import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";
import { themes as prismThemes } from "prism-react-renderer";

const siteUrl = process.env["DOCS_SITE_URL"] ?? "http://localhost:4300";
const baseUrl = process.env["DOCS_BASE_URL"] ?? "/";
const appUrl = process.env["DOCS_APP_URL"] ?? "http://localhost:4200";
const appLabel = process.env["DOCS_APP_LABEL"] ?? "Mở DX-OS";

const config: Config = {
  title: "DX-OS Docs",
  tagline: "Tài liệu vận hành, phát triển và kiến trúc DX-OS",
  favicon: "img/favicon.svg",
  url: siteUrl,
  baseUrl,
  organizationName: "tungnguyen2k5vp",
  projectName: "DX-OS",
  trailingSlash: false,
  onBrokenLinks: "throw",
  markdown: {
    format: "detect",
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: "throw",
    },
  },
  i18n: {
    defaultLocale: "vi",
    locales: ["vi"],
    localeConfigs: {
      vi: {
        label: "Tiếng Việt",
        htmlLang: "vi-VN",
      },
    },
  },
  presets: [
    [
      "classic",
      {
        docs: {
          path: "../docs",
          routeBasePath: "/",
          sidebarPath: "./sidebars.ts",
          showLastUpdateTime: true,
          breadcrumbs: true,
        },
        blog: false,
        sitemap: {
          changefreq: "weekly",
          priority: 0.5,
        },
        theme: {
          customCss: "./src/css/custom.css",
        },
      } satisfies Preset.Options,
    ],
  ],
  themes: ["@docusaurus/theme-mermaid"],
  plugins: [
    [
      "@easyops-cn/docusaurus-search-local",
      {
        hashed: true,
        indexDocs: true,
        indexBlog: false,
        indexPages: true,
        docsRouteBasePath: "/",
        docsDir: "../docs",
        removeDefaultStopWordFilter: true,
        removeDefaultStemmer: true,
        highlightSearchTermsOnTargetPage: true,
        explicitSearchResultPath: true,
      },
    ],
  ],
  themeConfig: {
    image: "img/dx-os-social-card.svg",
    metadata: [
      {
        name: "description",
        content:
          "Tài liệu cài đặt, sử dụng, kiến trúc, API và vận hành DX-OS Lab.",
      },
    ],
    announcementBar: {
      id: "mvp-status",
      content:
        "DX-OS hiện đã có Procurement, Budget, Attachments và Reporting. RAG/Agent đang ở lộ trình tiếp theo.",
      backgroundColor: "#ccfbf1",
      textColor: "#134e4a",
      isCloseable: true,
    },
    colorMode: {
      defaultMode: "light",
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: "DX-OS Docs",
      logo: {
        alt: "DX-OS",
        src: "img/logo.svg",
      },
      hideOnScroll: false,
      items: [
        { to: "/", label: "Trang chủ", position: "left", exact: true },
        { to: "/bat-dau", label: "Bắt đầu", position: "left" },
        {
          to: "/huong-dan-su-dung",
          label: "Hướng dẫn sử dụng",
          position: "left",
        },
        { to: "/architecture/CONTEXT", label: "Kiến trúc", position: "left" },
        { type: "search", position: "right" },
        {
          href: appUrl,
          label: appLabel,
          position: "right",
          className: "navbar-app-link",
        },
      ],
    },
    footer: {
      style: "dark",
      links: [
        {
          title: "Bắt đầu",
          items: [
            { label: "Cài đặt local", to: "/bat-dau" },
            { label: "Hướng dẫn sử dụng", to: "/huong-dan-su-dung" },
            {
              label: "Xử lý sự cố",
              to: "/huong-dan-su-dung#14-xử-lý-sự-cố-cho-người-dùng",
            },
          ],
        },
        {
          title: "Kỹ thuật",
          items: [
            { label: "Kiến trúc", to: "/architecture/CONTEXT" },
            { label: "REST API", to: "/implementation/API" },
            { label: "Authentication", to: "/implementation/AUTHORIZATION" },
          ],
        },
        {
          title: "Dịch vụ local",
          items: [
            { label: "DX-OS", href: "http://localhost:4200" },
            { label: "Keycloak", href: "http://localhost:8080" },
            { label: "Metabase", href: "http://localhost:3000" },
          ],
        },
      ],
      copyright: "DX-OS Lab · Tài liệu dành cho môi trường học tập và MVP.",
    },
    docs: {
      sidebar: {
        hideable: true,
        autoCollapseCategories: true,
      },
    },
    tableOfContents: {
      minHeadingLevel: 2,
      maxHeadingLevel: 3,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ["bash", "powershell", "json", "go"],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
