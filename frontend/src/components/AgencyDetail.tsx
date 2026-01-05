import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { api, AgencyDetail as AgencyDetailType, Title, ChecksumInfo, TitleBreakdown, Agency } from '../api';
import Layout from './Layout';

const AgencyDetail: React.FC = () => {
  const { slug } = useParams<{ slug: string }>();
  const [agency, setAgency] = useState<AgencyDetailType | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [latestDate, setLatestDate] = useState<string | null>(null);
  const [titles, setTitles] = useState<Title[]>([]);
  const [checksums, setChecksums] = useState<ChecksumInfo[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      if (!slug) return;
      
      try {
        setLoading(true);
        
        // Fetch agency details, titles, and checksums
        const [agencyResponse, titlesResponse, checksumsResponse] = await Promise.all([
          api.getAgencyDetail(slug),
          api.getTitles(),
          api.getChecksums()
        ]);
        
        setAgency(agencyResponse.data);
        setTitles(titlesResponse.data);
        setChecksums(checksumsResponse.data);
        
        // Use the latest issue date from our database
        setLatestDate(titlesResponse.meta.lastUpdated.split('T')[0]);
        
      } catch (err) {
        console.error('Failed to fetch agency details:', err);
        setError(err instanceof Error ? err.message : 'Failed to load agency details');
      } finally {
        setLoading(false);
      }
    };


    fetchData();
  }, [slug]);

  if (loading) {
    return (
      <Layout>
        <div className="text-center padding-4">
          <p className="font-body-lg">Loading agency details...</p>
        </div>
      </Layout>
    );
  }

  if (error || !agency) {
    return (
      <Layout>
        <div className="usa-alert usa-alert--error">
          <div className="usa-alert__body">
            <h3 className="usa-alert__heading">Error Loading Agency Details</h3>
            <p className="usa-alert__text">{error || 'Agency not found'}</p>
            <Link to="/dashboard" className="usa-button usa-button--outline">
              Back to Dashboard
            </Link>
          </div>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="agency-detail">
        {/* Breadcrumb */}
        <nav className="usa-breadcrumb" aria-label="Breadcrumbs">
          <ol className="usa-breadcrumb__list">
            <li className="usa-breadcrumb__list-item">
              <Link to="/dashboard" className="usa-breadcrumb__link">
                Dashboard
              </Link>
            </li>
            <li className="usa-breadcrumb__list-item usa-current" aria-current="page">
              <span>{agency.name}</span>
            </li>
          </ol>
        </nav>

        {/* Header */}
        <div className="margin-bottom-4 margin-top-3">
          <h1 className="font-heading-2xl margin-bottom-2">{agency.name}</h1>
          <p className="font-body-lg text-base-dark">
            Agency Code: <strong>{agency.slug}</strong>
          </p>
        </div>

        {/* Summary Stats */}
        <div className="margin-bottom-5">
          <div className="usa-card">
            <div className="usa-card__container">
              <div className="usa-card__header">
                <h2 className="usa-card__heading">Agency Summary</h2>
              </div>
              <div className="usa-card__body">
                <div className="grid-row grid-gap">
                  <div className="grid-col-12 tablet:grid-col-4">
                    <div className="stat-box">
                      <div className="stat-value">{agency.wordCount.toLocaleString()}</div>
                      <div className="stat-label">Total Word Count</div>
                    </div>
                  </div>
                  <div className="grid-col-12 tablet:grid-col-4">
                    <div className="stat-box">
                      <div className="stat-value">{agency.titleCount}</div>
                      <div className="stat-label">CFR Titles Regulated</div>
                    </div>
                  </div>
                  <div className="grid-col-12 tablet:grid-col-4">
                    <div className="stat-box">
                      <div className="stat-value">{agency.subAgencies?.length || 0}</div>
                      <div className="stat-label">Sub-Agencies</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Agency Checksum */}
        {agency.checksum && (
          <div className="margin-bottom-5">
            <h2 className="font-heading-xl margin-bottom-3">Agency Checksum</h2>
            <div className="usa-card">
              <div className="usa-card__container">
                <div className="usa-card__body">
                  <div className="grid-row">
                    <div className="grid-col-12">
                      <p className="margin-bottom-2">
                        <strong>Data integrity hash for all CFR titles regulated by this agency:</strong>
                      </p>
                      <div className="checksum-full-display">
                        <code className="checksum-hash">
                          {agency.checksum}
                        </code>
                        <button
                          type="button"
                          className="usa-button usa-button--small usa-button--outline margin-left-2"
                          onClick={() => navigator.clipboard.writeText(agency.checksum || '')}
                          title="Copy checksum to clipboard"
                        >
                          Copy
                        </button>
                      </div>
                      <p className="margin-top-2 text-base-dark font-body-sm">
                        This checksum represents the combined regulatory content under this agency's purview. 
                        If any CFR title regulated by this agency changes, this checksum will change too.
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Title Breakdown */}
        {agency.titleBreakdown && agency.titleBreakdown.length > 0 && (
          <div className="margin-bottom-5">
            <h2 className="font-heading-xl margin-bottom-3">CFR Title Breakdown</h2>
            <div className="usa-card">
              <div className="usa-card__container">
                <div className="usa-card__body">
                  <div className="usa-table-container--scrollable">
                    <table className="usa-table usa-table--striped">
                      <thead>
                        <tr>
                          <th scope="col">Title Number</th>
                          <th scope="col">Title Name</th>
                          <th scope="col" className="text-right">Word Count</th>
                          <th scope="col" className="text-right">% of Agency Total</th>
                          <th scope="col">Checksum</th>
                          <th scope="col">Download</th>
                        </tr>
                      </thead>
                      <tbody>
                        {agency.titleBreakdown.map((title: TitleBreakdown, index: number) => {
                          const percentage = agency.wordCount > 0 
                            ? (title.wordCount / agency.wordCount * 100).toFixed(2)
                            : '0.00';
                          
                          // Find checksum for this title
                          const checksumInfo = checksums.find((c: ChecksumInfo) => c.titleNumber === title.titleNumber);
                          
                          const handleDownload = () => {
                            // Find the title data from our database to get the latest date
                            const titleData = titles.find((t: Title) => t.number === title.titleNumber);
                            const titleLatestDate = titleData?.upToDateAsOf 
                              ? titleData.upToDateAsOf.split('T')[0] 
                              : latestDate;
                            
                            if (!titleLatestDate) {
                              alert('Latest date not yet determined. Please try again in a moment.');
                              return;
                            }
                            const downloadUrl = `https://www.ecfr.gov/api/versioner/v1/full/${titleLatestDate}/title-${title.titleNumber}.xml`;
                            window.open(downloadUrl, '_blank');
                          };
                          
                          return (
                            <tr key={index}>
                              <th scope="row">Title {title.titleNumber}</th>
                              <td>{title.titleName}</td>
                              <td className="text-right font-mono-sm">
                                {title.wordCount.toLocaleString()}
                              </td>
                              <td className="text-right font-mono-sm">
                                {percentage}%
                              </td>
                              <td className="checksum-cell">
                                {checksumInfo?.checksum ? (
                                  <span 
                                    className="checksum-display font-mono-xs"
                                    title={`Full checksum: ${checksumInfo.checksum}\nLast changed: ${new Date(checksumInfo.lastChanged).toLocaleDateString()}`}
                                  >
                                    {checksumInfo.checksum.substring(0, 8)}...
                                  </span>
                                ) : (
                                  <span className="text-base-light">No checksum</span>
                                )}
                              </td>
                              <td>
                                <button
                                  type="button"
                                  className="usa-button usa-button--small usa-button--outline"
                                  onClick={handleDownload}
                                  title={`Download Title ${title.titleNumber} XML`}
                                >
                                  Download XML
                                </button>
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Sub-Agencies */}
        {agency.subAgencies && agency.subAgencies.length > 0 && (
          <div className="margin-bottom-5">
            <h2 className="font-heading-xl margin-bottom-3">Sub-Agencies</h2>
            <div className="usa-card">
              <div className="usa-card__container">
                <div className="usa-card__body">
                  <div className="usa-table-container--scrollable">
                    <table className="usa-table usa-table--striped">
                      <thead>
                        <tr>
                          <th scope="col">Agency Name</th>
                          <th scope="col" className="text-right">Word Count</th>
                          <th scope="col" className="text-right">% of Total CFR</th>
                          <th scope="col" className="text-right">Titles</th>
                        </tr>
                      </thead>
                      <tbody>
                        {agency.subAgencies.map((subAgency: Agency) => (
                          <tr key={subAgency.id}>
                            <th scope="row">
                              <Link 
                                to={`/agency/${subAgency.slug}`}
                                className="usa-link"
                              >
                                {subAgency.name}
                              </Link>
                            </th>
                            <td className="text-right font-mono-sm">
                              {subAgency.wordCount.toLocaleString()}
                            </td>
                            <td className="text-right font-mono-sm">
                              {subAgency.percentOfTotal.toFixed(2)}%
                            </td>
                            <td className="text-right font-mono-sm">
                              {subAgency.titleCount}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Back Button */}
        <div className="margin-top-5">
          <Link to="/dashboard" className="usa-button usa-button--outline">
            ← Back to Dashboard
          </Link>
        </div>
      </div>

      <style>{`
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
        
        .font-mono-sm {
          font-family: 'Courier New', monospace;
          font-size: 0.875rem;
        }
        
        .font-mono-xs {
          font-family: 'Courier New', monospace;
          font-size: 0.75rem;
        }
        
        .text-right {
          text-align: right;
        }
        
        .usa-table-container--scrollable {
          overflow-x: auto;
        }
        
        .agency-detail {
          margin-bottom: 4rem;
        }
        
        .checksum-cell {
          max-width: 120px;
        }
        
        .checksum-display {
          cursor: help;
          color: #005ea2;
          border-bottom: 1px dotted #005ea2;
        }
        
        .checksum-display:hover {
          background-color: #f0f9ff;
        }
        
        .checksum-full-display {
          display: flex;
          align-items: center;
          flex-wrap: wrap;
          gap: 0.5rem;
        }
        
        .checksum-hash {
          font-family: 'Courier New', monospace;
          font-size: 0.875rem;
          background-color: #f0f9ff;
          padding: 0.75rem;
          border-radius: 0.25rem;
          border: 1px solid #bfdbfe;
          word-break: break-all;
          flex: 1;
          min-width: 300px;
        }
      `}</style>
    </Layout>
  );
};

export default AgencyDetail;