import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

// The search theme's own PluginOptions type isn't assignable to Docusaurus's
// generic PluginConfig tuple — cast the pair when mounting it.
const searchLocal: [string, object] = [
  '@easyops-cn/docusaurus-search-local',
  {
    hashed: true,
    docsRouteBasePath: '/docs',
    docsDir: ['../docs'],
    indexBlog: false,
    indexPages: false,
  },
];

const config: Config = {
  title: 'Thoth',
  tagline:
    'Your local-first personal knowledge base — plain markdown you own, a built-in assistant that files it for you.',
  favicon: 'img/favicon.svg',

  // Deploy URL placeholder — the site is served locally via `make docs-dev`;
  // wire these up when a hosting destination is chosen.
  url: 'https://thoth-docs.local',
  baseUrl: '/',
  organizationName: 'shiv-source',
  projectName: 'thoth',

  onBrokenLinks: 'throw',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          path: '../docs',
          routeBasePath: '/docs',
          editUrl: 'https://github.com/shiv-source/thoth/edit/main/docs',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  // Mermaid diagrams in markdown (theme-mermaid renders the fenced blocks).
  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: ({sourceFilePath, url}) => {
        console.error(`Broken markdown link in ${sourceFilePath}: ${url}`);
        throw new Error(`Found broken markdown link: ${url}`);
      },
    },
  },

  themes: ['@docusaurus/theme-mermaid', searchLocal],

  themeConfig: {
    image: 'img/social-card.svg',
    colorMode: {
      defaultMode: 'light',
      disableSwitch: true,
      respectPrefersColorScheme: false,
    },
    mermaid: {
      theme: {light: 'base', dark: 'base'},
    },
    navbar: {
      title: 'Thoth',
      logo: {
        alt: 'Thoth',
        src: 'img/logo.svg',
      },
      items: [
        {to: '/docs/', label: 'Docs', position: 'left'},
        {to: '/docs/getting-started', label: 'Getting started', position: 'left'},
        {to: '/docs/using-thoth', label: 'Using Thoth', position: 'left'},
        {
          href: 'https://github.com/shiv-source/thoth',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'light',
      links: [
        {
          title: 'Docs',
          items: [
            {label: 'Getting started', to: '/docs/getting-started'},
            {label: 'Using Thoth', to: '/docs/using-thoth'},
            {label: 'Architecture', to: '/docs/architecture'},
            {label: 'Troubleshooting', to: '/docs/troubleshooting'},
          ],
        },
        {
          title: 'Reference',
          items: [
            {label: 'CLI', to: '/docs/cli'},
            {label: 'API', to: '/docs/api'},
            {label: 'Schema', to: '/docs/schema'},
            {label: 'Security', to: '/docs/security'},
          ],
        },
        {
          title: 'Project',
          items: [
            {label: 'GitHub', href: 'https://github.com/shiv-source/thoth'},
            {
              label: 'Contribute',
              to: '/docs/development',
            },
            {label: 'License', href: 'https://github.com/shiv-source/thoth/blob/main/LICENSE'},
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Thoth contributors.`,
    },
    prism: {
      theme: prismThemes.github,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
