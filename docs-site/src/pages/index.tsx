import React from 'react';
import useBaseUrl from '@docusaurus/useBaseUrl';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';

import styles from './index.module.css';

type Feature = {
  title: string;
  description: string;
};

const features: Feature[] = [
  {
    title: 'You own your knowledge',
    description:
      'Everything you know lives as plain markdown in one directory you control — readable, diffable, portable. No proprietary format, no lock-in.',
  },
  {
    title: 'Ask, don\'t search',
    description:
      'Chat with your wiki in natural language. A built-in assistant answers from your notes and saves new knowledge back into the right place.',
  },
  {
    title: 'Organized by default',
    description:
      'Meeting notes, projects, links, setups, and TODOs each have a home. A rulebook teaches the assistant how to file and find everything.',
  },
  {
    title: 'Local-first and private',
    description:
      'One static binary, SQLite full-text search, localhost only. No cloud, no account, nothing leaves your machine.',
  },
];

function FeatureCard({title, description}: Feature) {
  return (
    <div className={styles.feature}>
      <h3>{title}</h3>
      <p>{description}</p>
    </div>
  );
}

function QuickStart() {
  const logo = useBaseUrl('/img/logo.svg');
  return (
    <section className={styles.quickStart}>
      <div className={styles.quickStartInner}>
        <div className={styles.terminal}>
          <div className={styles.terminalHeader}>
            <span>terminal</span>
          </div>
          <pre>
            <code>{`$ thoth serve
→ http://127.0.0.1:8333
$ thoth doctor
→ all checks healthy`}</code>
          </pre>
        </div>
        <div className={styles.quickStartText}>
          <h2>Up and running in minutes</h2>
          <p>
            Install Claude Code, then start the server — it scaffolds your wiki
            automatically on first run. Open the dashboard and ask{' '}
            <em>"what did we decide in Tuesday's standup?"</em> — or say{' '}
            <em>"save this: …"</em> and watch it get filed, searchable within
            seconds.
          </p>
          <div className={styles.quickStartLinks}>
            <Link className={styles.buttonPrimary} to="/docs/getting-started">
              Get started
            </Link>
            <Link className={styles.buttonGhost} to="/docs/architecture">
              Read the architecture
            </Link>
          </div>
        </div>
        <img src={logo} alt="" className={styles.hiddenMark} />
      </div>
    </section>
  );
}

export default function Home(): React.JSX.Element {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title="Local-first personal knowledge base"
      description={siteConfig.tagline}
    >
      <header className={styles.hero}>
        <div className={styles.heroInner}>
          <h1 className={styles.heroTitle}>Your knowledge, as files you own.</h1>
          <p className={styles.heroSubtitle}>
            {siteConfig.tagline}
          </p>
          <div className={styles.heroLinks}>
            <Link className={styles.buttonPrimary} to="/docs/getting-started">
              Get started
            </Link>
            <Link className={styles.buttonGhost} to="/docs/using-thoth">
              Using Thoth
            </Link>
          </div>
          <div className={styles.heroPills}>
            <span>One binary</span>
            <span>No cloud</span>
            <span>Localhost only</span>
          </div>
        </div>
      </header>

      <main>
        <section className={styles.features}>
          {features.map((f) => (
            <FeatureCard key={f.title} {...f} />
          ))}
        </section>

        <QuickStart />

        <section className={styles.explore}>
          <h2>Explore the docs</h2>
          <div className={styles.exploreGrid}>
            <Link to="/docs/using-thoth" className={styles.exploreCard}>
              <h3>Using Thoth</h3>
              <p>
                The dashboard tour, chat, search, settings, GitHub sync, and
                best practices for getting the most out of your wiki.
              </p>
            </Link>
            <Link to="/docs/architecture" className={styles.exploreCard}>
              <h3>Architecture</h3>
              <p>
                The two layers, the data contract, and how one binary drives
                everything — with Mermaid diagrams.
              </p>
            </Link>
            <Link to="/docs/knowledge-base" className={styles.exploreCard}>
              <h3>Knowledge base</h3>
              <p>
                The wiki layout, conventions, and the rulebook that keeps notes
                filed consistently.
              </p>
            </Link>
            <Link to="/docs/troubleshooting" className={styles.exploreCard}>
              <h3>Troubleshooting</h3>
              <p>
                Common issues and fixes — from missing binaries to an
                out-of-sync index.
              </p>
            </Link>
          </div>
        </section>
      </main>
    </Layout>
  );
}
