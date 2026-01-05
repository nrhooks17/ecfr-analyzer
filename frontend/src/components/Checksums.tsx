import React, { useEffect, useState } from 'react';
import { api, AgencyChecksumInfo } from '../api';
import Layout from './Layout';

const Checksums: React.FC = () => {
  const [checksums, setChecksums] = useState<AgencyChecksumInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchChecksums = async () => {
      try {
        const response = await api.getAgencyChecksums();
        setChecksums(response.data);
      } catch (err) {
        console.error('Failed to fetch checksums:', err);
        setError(err instanceof Error ? err.message : 'Failed to load checksums');
      } finally {
        setLoading(false);
      }
    };

    fetchChecksums();
  }, []);

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text).then(() => {
      // You could add a toast notification here
    });
  };

  if (loading) {
    return (
      <Layout>
        <div className="text-center padding-4">
          <p className="font-body-lg">Loading checksums...</p>
        </div>
      </Layout>
    );
  }

  if (error) {
    return (
      <Layout>
        <div className="usa-alert usa-alert--error">
          <div className="usa-alert__body">
            <h3 className="usa-alert__heading">Error Loading Checksums</h3>
            <p className="usa-alert__text">{error}</p>
          </div>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="checksums">
        <div className="margin-bottom-4">
          <h1 className="font-heading-2xl margin-bottom-2">CFR Agency Checksums</h1>
          <p className="font-body-lg text-base-dark">
            Data integrity verification for all CFR agencies
          </p>
        </div>

        {/* Info Alert */}
        <div className="usa-alert usa-alert--info margin-bottom-4">
          <div className="usa-alert__body">
            <h4 className="usa-alert__heading">About Checksums</h4>
            <p className="usa-alert__text">
              Checksums are cryptographic hashes that verify data integrity. Each agency's content 
              generates a unique checksum - if the content changes, the checksum changes too. 
              This helps detect corruption and track when regulations are updated.
            </p>
          </div>
        </div>

        {/* Checksums Table */}
        <div className="usa-card">
          <div className="usa-card__container">
            <div className="usa-card__header">
              <h2 className="usa-card__heading">All CFR Agency Checksums</h2>
            </div>
            <div className="usa-card__body">
              <div className="usa-table-container--scrollable">
                <table className="usa-table usa-table--striped">
                  <thead>
                    <tr>
                      <th scope="col">Agency</th>
                      <th scope="col">Name</th>
                      <th scope="col">Checksum</th>
                      <th scope="col">Last Changed</th>
                      <th scope="col">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {checksums.map((checksum: AgencyChecksumInfo) => (
                      <tr key={checksum.agencyId}>
                        <th scope="row">{checksum.agencySlug}</th>
                        <td className="agency-name-col">{checksum.agencyName}</td>
                        <td className="checksum-full">
                          {checksum.checksum ? (
                            <code className="checksum-hash">
                              {checksum.checksum}
                            </code>
                          ) : (
                            <span className="text-base-light">No checksum available</span>
                          )}
                        </td>
                        <td className="font-mono-sm">
                          {new Date(checksum.lastChanged).toLocaleDateString()}
                        </td>
                        <td>
                          {checksum.checksum && (
                            <button
                              type="button"
                              className="usa-button usa-button--small usa-button--outline"
                              onClick={() => copyToClipboard(checksum.checksum!)}
                              title="Copy checksum to clipboard"
                            >
                              Copy
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>

        {/* Summary Stats */}
        <div className="margin-top-4">
          <div className="usa-card">
            <div className="usa-card__container">
              <div className="usa-card__header">
                <h3 className="usa-card__heading">Summary</h3>
              </div>
              <div className="usa-card__body">
                <div className="grid-row grid-gap">
                  <div className="grid-col-12 tablet:grid-col-4">
                    <div className="stat-box">
                      <div className="stat-value">{checksums.length}</div>
                      <div className="stat-label">Total Agencies</div>
                    </div>
                  </div>
                  <div className="grid-col-12 tablet:grid-col-4">
                    <div className="stat-box">
                      <div className="stat-value">
                        {checksums.filter((c: AgencyChecksumInfo) => c.checksum).length}
                      </div>
                      <div className="stat-label">With Checksums</div>
                    </div>
                  </div>
                  <div className="grid-col-12 tablet:grid-col-4">
                    <div className="stat-box">
                      <div className="stat-value">
                        {checksums.filter((c: AgencyChecksumInfo) => !c.checksum).length}
                      </div>
                      <div className="stat-label">Missing Checksums</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <style>{`
        .checksums {
          margin-bottom: 4rem;
        }
        
        .font-mono-sm {
          font-family: 'Courier New', monospace;
          font-size: 0.875rem;
        }
        
        .usa-table-container--scrollable {
          overflow-x: auto;
        }
        
        .agency-name-col {
          max-width: 200px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
        
        .checksum-full {
          min-width: 480px;
          max-width: 600px;
          word-break: break-all;
        }
        
        .checksum-hash {
          font-family: 'Courier New', monospace;
          font-size: 0.75rem;
          background-color: #f0f9ff;
          padding: 0.25rem 0.5rem;
          border-radius: 0.25rem;
          border: 1px solid #bfdbfe;
          display: block;
          word-break: break-all;
        }
        
        .stat-box {
          text-align: center;
          padding: 1.5rem;
          background-color: #f8f9fa;
          border-radius: 0.5rem;
          border: 1px solid #dee2e6;
        }
        
        .stat-value {
          font-size: 2rem;
          font-weight: 700;
          color: #005ea2;
          line-height: 1.1;
          margin-bottom: 0.5rem;
        }
        
        .stat-label {
          font-size: 0.875rem;
          font-weight: 600;
          color: #1b1b1b;
          text-transform: uppercase;
          letter-spacing: 0.025em;
        }
      `}</style>
    </Layout>
  );
};

export default Checksums;