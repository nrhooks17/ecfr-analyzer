import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { api, Agency, WordCountMetrics, Title } from '../api';
import Layout from './Layout';
import SummaryCard from './SummaryCard';
import AgencyTable from './AgencyTable';
import ExportButton from './ExportButton';

const Dashboard: React.FC = () => {
  const navigate = useNavigate();
  const [metrics, setMetrics] = useState<WordCountMetrics | null>(null);
  const [titles, setTitles] = useState<Title[]>([]);
  const [agencies, setAgencies] = useState<Agency[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<string>('');

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [metricsResponse, titlesResponse, agenciesResponse] = await Promise.all([
          api.getWordCountMetrics(),
          api.getTitles(),
          api.getAgencies(),
        ]);

        setMetrics(metricsResponse.data);
        setTitles(titlesResponse.data);
        setAgencies(agenciesResponse.data);
        setLastUpdated(metricsResponse.meta.lastUpdated);
      } catch (err) {
        console.error('Failed to fetch dashboard data:', err);
        setError(err instanceof Error ? err.message : 'Failed to load data');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  if (loading) {
    return (
      <Layout>
        <div className="text-center padding-4">
          <p className="font-body-lg">Loading dashboard...</p>
        </div>
      </Layout>
    );
  }

  if (error) {
    return (
      <Layout>
        <div className="usa-alert usa-alert--error">
          <div className="usa-alert__body">
            <h3 className="usa-alert__heading">Error Loading Dashboard</h3>
            <p className="usa-alert__text">{error}</p>
          </div>
        </div>
      </Layout>
    );
  }

  const activeTitles = titles.filter((title: Title) => title.wordCount && title.wordCount > 0);
  const topAgencies = agencies.slice(0, 10);

  // Prepare data for bar chart
  const chartData = topAgencies.map((agency: Agency) => ({
    name: agency.name.length > 20 ? agency.name.substring(0, 20) + '...' : agency.name,
    fullName: agency.name,
    wordCount: agency.wordCount,
    percentage: agency.percentOfTotal,
  }));


  const handleAgencyClick = (agency: Agency) => {
    navigate(`/agency/${agency.slug}`);
  };

  return (
    <Layout>
      <div className="dashboard">
        <div className="grid-row margin-bottom-4">
          <div className="grid-col">
            <h1 className="font-heading-2xl margin-bottom-2">eCFR Analysis Dashboard</h1>
            <p className="font-body-lg text-base-dark">
              Comprehensive analysis of the electronic Code of Federal Regulations
            </p>
          </div>
          <div className="grid-col-auto">
            <ExportButton 
              type="metrics" 
              label="Export All Data"
              className="usa-button usa-button--big"
            />
          </div>
        </div>

        {/* Summary Card */}
        <div className="margin-bottom-5">
          <div className="usa-card">
            <div className="usa-card__container">
              <div className="usa-card__body">
                <div className="summary-stats">
                  <div className="stat-item">
                    <span className="stat-label">Total CFR Words:</span>
                    <span className="stat-value">{(metrics?.totalCFRWords || 0).toLocaleString()}</span>
                  </div>
                  <div className="stat-item">
                    <span className="stat-label">Active Agencies:</span>
                    <span className="stat-value">{agencies.length}</span>
                  </div>
                  <div className="stat-item">
                    <span className="stat-label">CFR Titles:</span>
                    <span className="stat-value">{activeTitles.length}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Top 10 Agencies Chart */}
        <div className="margin-bottom-5">
          <h2 className="font-heading-xl margin-bottom-3">Top 10 Agencies by Word Count</h2>
          <div className="usa-card">
            <div className="usa-card__container">
              <div className="usa-card__body">
                <ResponsiveContainer width="100%" height={400}>
                  <BarChart
                    data={chartData}
                    margin={{ top: 20, right: 30, left: 20, bottom: 80 }}
                  >
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis 
                      dataKey="name" 
                      angle={-45}
                      textAnchor="end"
                      height={100}
                      fontSize={12}
                    />
                    <YAxis 
                      tickFormatter={(value: number) => `${(value / 1000000).toFixed(1)}M`}
                    />
                    <Tooltip 
                      formatter={(value: number | string) => [
                        `${value.toLocaleString()} words`,
                        'Word Count'
                      ]}
                      labelFormatter={(label: string, payload: Array<{ payload?: { fullName?: string } }>) => 
                        payload?.[0]?.payload?.fullName || label
                      }
                    />
                    <Bar dataKey="wordCount" fill="#005ea2" />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>
        </div>


        {/* Agencies Table */}
        <div className="margin-bottom-5">
          <div className="grid-row">
            <div className="grid-col">
              <h2 className="font-heading-xl margin-bottom-3">All Agencies</h2>
            </div>
            <div className="grid-col-auto">
              <ExportButton type="agencies" />
            </div>
          </div>
          
          <div className="usa-card">
            <div className="usa-card__container">
              <div className="usa-card__body">
                <AgencyTable 
                  agencies={agencies}
                  onAgencyClick={handleAgencyClick}
                />
              </div>
            </div>
          </div>
        </div>

        {/* Footer Info */}
        <div className="usa-alert usa-alert--info">
          <div className="usa-alert__body">
            <p className="usa-alert__text">
              Data is automatically refreshed every hour from the official eCFR API. 
              Last update: {new Date(lastUpdated).toLocaleString()}
            </p>
          </div>
        </div>
      </div>

      <style>{`
        .dashboard {
          margin-bottom: 4rem;
          padding-left: 0rem;
          padding-right: 0rem;
        }
        
        .summary-stats {
          display: flex;
          gap: 2rem;
          flex-wrap: wrap;
        }
        
        .stat-item {
          display: flex;
          gap: 0.5rem;
          align-items: baseline;
        }
        
        .stat-label {
          font-weight: 600;
          color: #1b1b1b;
        }
        
        .stat-value {
          font-weight: 700;
          font-size: 1.25rem;
          color: #005ea2;
        }
      `}</style>
    </Layout>
  );
};

export default Dashboard;