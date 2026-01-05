import React from 'react';
import { Link } from 'react-router-dom';

interface LayoutProps {
  children: React.ReactNode;
  showNavigation?: boolean;
}

const Layout: React.FC<LayoutProps> = ({ children, showNavigation = true }) => {
  return (
    <div className="usa-layout">
      {/* US Web Design System Banner */}
      <section className="usa-banner" aria-label="Official website">
        <div className="usa-accordion">
          <header className="usa-banner__header">
            <div className="usa-banner__inner">
              <div className="grid-col-auto">
                <img
                  aria-hidden="true"
                  className="usa-banner__header-flag"
                  src="https://cdn.jsdelivr.net/npm/@uswds/uswds@3.7.1/dist/img/us_flag_small.png"
                  alt=""
                />
              </div>
              <div className="grid-col-fill tablet:grid-col-auto" aria-hidden="true">
                <p className="usa-banner__header-text">
                  An official website of the United States government
                </p>
                <p className="usa-banner__header-action">Here's how you know</p>
              </div>
            </div>
          </header>
        </div>
      </section>

      {/* Header */}
      <header className="usa-header usa-header--basic">
        <div className="grid-container-widescreen">
          <div className="usa-navbar">
            <div className="usa-logo">
              <em className="usa-logo__text">
                <Link to="/" title="eCFR Analyzer">
                  eCFR Analyzer
                </Link>
              </em>
            </div>
            {showNavigation && (
              <nav aria-label="Primary navigation" className="usa-nav">
                <ul className="usa-nav__primary usa-accordion">
                  <li className="usa-nav__primary-item">
                    <Link className="usa-nav__link" to="/">
                      Dashboard
                    </Link>
                  </li>
                  <li className="usa-nav__primary-item">
                    <Link className="usa-nav__link" to="/trends">
                      Trends
                    </Link>
                  </li>
                  <li className="usa-nav__primary-item">
                    <Link className="usa-nav__link" to="/checksums">
                      Checksums
                    </Link>
                  </li>
                </ul>
              </nav>
            )}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="usa-section usa-prose">
        <div className="grid-container-widescreen">
          {children}
        </div>
      </main>
    </div>
  );
};

export default Layout;