import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

// The content is the repo's `docs/` directory (flat markdown files — see
// docusaurus.config.ts: docs.path = '../docs'). The sidebar groups those
// flat files into named categories by doc id (filename without extension).
const sidebars: SidebarsConfig = {
  docs: [
    {
      type: 'category',
      label: 'Getting started',
      collapsed: false,
      items: ['whats-new', 'index', 'getting-started'],
    },
    {
      type: 'category',
      label: 'Using Thoth',
      collapsed: false,
      items: ['using-thoth', 'browser-extension'],
    },
    {
      type: 'category',
      label: 'Core concepts',
      items: ['thoth-agent', 'architecture', 'knowledge-base'],
    },
    {
      type: 'category',
      label: 'Reference',
      items: ['cli', 'api', 'schema', 'indexing', 'components', 'frontend', 'security'],
    },
    {
      type: 'category',
      label: 'Contributing & development',
      items: ['development'],
    },
    'troubleshooting',
  ],
};

export default sidebars;
