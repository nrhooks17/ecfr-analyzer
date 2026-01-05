import React, { useEffect, useState } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { api, Agency, HistoricalPoint } from '../api';
import Layout from './Layout';

const Trends: React.FC = () => {
  const [agencies, setAgencies] = useState<Agency[]>([]);
  const [selectedAgency, setSelectedAgency] = useState<string>('');
  const [timeRange, setTimeRange] = useState<number>(12);
  const [historyData, setHistoryData] = useState<HistoricalPoint[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchAgencies = async () => {
      try {
        const response = await api.getAgencies();
        setAgencies(response.data);
      } catch (err) {
        console.error('Failed to fetch agencies:', err);
        setError(err instanceof Error ? err.message : 'Failed to load agencies');
      }
    };

    fetchAgencies();
  }, []);

  useEffect(() => {
    const fetchHistory = async () => {
      setLoading(true);
      setError(null);

      try {
        const response = await api.getHistory(selectedAgency || undefined, timeRange);
        setHistoryData(response.data);
      } catch (err) {
        console.error('Failed to fetch history:', err);
        setError(err instanceof Error ? err.message : 'Failed to load historical data');
      } finally {
        setLoading(false);
      }
    };

    fetchHistory();
  }, [selectedAgency, timeRange]);

  const handleAgencyChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    setSelectedAgency(event.target.value);
  };

  const handleTimeRangeChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    setTimeRange(parseInt(event.target.value, 10));
  };

  const chartData = (historyData || []).map((point: HistoricalPoint) => ({
    date: new Date(point.date).toLocaleDateString(),
    wordCount: point.wordCount,
    change: point.changePercent,
  }));

  const selectedAgencyData = selectedAgency 
    ? agencies.find((a: Agency) => a.slug === selectedAgency)
    : null;

  return (
    <Layout>
      <div className="trends">
        <div className="margin-bottom-4">
          <h1 className="font-heading-2xl margin-bottom-2">Historical Trends</h1>
          <p className="font-body-lg text-base-dark">
            Track regulatory growth and changes over time
          </p>
        </div>

        {/* Controls */}
        <div className="usa-card margin-bottom-4">
          <div className="usa-card__container">
            <div className="usa-card__body">
              <div className="grid-row grid-gap">
                <div className="grid-col-12 tablet:grid-col-6">
                  <label htmlFor="agency-select" className="usa-label">
                    Select Agency
                  </label>
                  <select
                    id="agency-select"
                    className="usa-select"
                    value={selectedAgency}
                    onChange={handleAgencyChange}
                  >
                    <option value="">All Agencies</option>
                    {agencies.map((agency: Agency) => (
                      <option key={agency.id} value={agency.slug}>
                        {agency.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="grid-col-12 tablet:grid-col-6">
                  <label htmlFor="time-range-select" className="usa-label">
                    Time Range
                  </label>
                  <select
                    id="time-range-select"
                    className="usa-select"
                    value={timeRange}
                    onChange={handleTimeRangeChange}
                  >
                    <option value={6}>Last 6 months</option>
                    <option value={12}>Last 12 months</option>
                    <option value={24}>Last 2 years</option>
                    <option value={60}>Last 5 years</option>
                    <option value={-1}>All time</option>
                  </select>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Selected Agency Info */}
        {selectedAgencyData && (
          <div className="usa-alert usa-alert--info margin-bottom-4">
            <div className="usa-alert__body">
              <h4 className="usa-alert__heading">{selectedAgencyData.name}</h4>
              <p className="usa-alert__text">
                Current word count: <strong>{selectedAgencyData.wordCount.toLocaleString()}</strong> 
                ({selectedAgencyData.percentOfTotal.toFixed(2)}% of total CFR)
              </p>
            </div>
          </div>
        )}

        {/* Chart */}
        <div className="margin-bottom-5">
          <h2 className="font-heading-xl margin-bottom-3">
            Word Count Over Time
            {selectedAgencyData && ` - ${selectedAgencyData.name}`}
          </h2>
          
          <div className="usa-card">
            <div className="usa-card__container">
              <div className="usa-card__body">
                {loading ? (
                  <div className="text-center padding-4">
                    <p>Loading historical data...</p>
                  </div>
                ) : error ? (
                  <div className="usa-alert usa-alert--error">
                    <div className="usa-alert__body">
                      <p className="usa-alert__text">{error}</p>
                    </div>
                  </div>
                ) : chartData.length === 0 ? (
                  <div className="usa-alert usa-alert--warning">
                    <div className="usa-alert__body">
                      <h4 className="usa-alert__heading">No Historical Data Available</h4>
                      <p className="usa-alert__text">
                        Historical trend data is not yet available. The system needs to collect data over time 
                        to show meaningful trends. Check back after the system has been running for a while.
                      </p>
                    </div>
                  </div>
                ) : (
                  <ResponsiveContainer width="100%" height={400}>
                    <LineChart data={chartData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="date" />
                      <YAxis tickFormatter={(value: number) => `${(value / 1000000).toFixed(1)}M`} />
                      <Tooltip 
                        formatter={(value: number | string) => [`${value.toLocaleString()} words`, 'Word Count']}
                      />
                      <Line type="monotone" dataKey="wordCount" stroke="#005ea2" strokeWidth={2} />
                    </LineChart>
                  </ResponsiveContainer>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Historical Table */}
        {chartData.length > 0 && (
          <div className="margin-bottom-5">
            <h3 className="font-heading-lg margin-bottom-3">Change History</h3>
            <div className="usa-card">
              <div className="usa-card__container">
                <div className="usa-card__body">
                  <div className="usa-table-container--scrollable">
                    <table className="usa-table">
                      <thead>
                        <tr>
                          <th scope="col">Date</th>
                          <th scope="col">Word Count</th>
                          <th scope="col">Change from Previous</th>
                        </tr>
                      </thead>
                      <tbody>
                        {historyData.map((point, index) => (
                          <tr key={index}>
                            <th scope="row">
                              {new Date(point.date).toLocaleDateString()}
                            </th>
                            <td className="font-mono-sm">
                              {point.wordCount.toLocaleString()}
                            </td>
                            <td className={`font-mono-sm ${
                              point.changePercent > 0 ? 'text-success' :
                              point.changePercent < 0 ? 'text-error' : ''
                            }`}>
                              {point.changePercent > 0 ? '+' : ''}
                              {point.changePercent.toFixed(2)}%
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

        {/* Info Box */}
        <div className="usa-alert usa-alert--info">
          <div className="usa-alert__body">
            <h4 className="usa-alert__heading">About Historical Data</h4>
            <p className="usa-alert__text">
              Historical snapshots are captured automatically to track regulatory changes over time. 
              This helps identify trends in regulatory growth, major updates, and agency activity patterns.
            </p>
          </div>
        </div>
      </div>

      <style>{`
        .font-mono-sm {
          font-family: 'Courier New', monospace;
          font-size: 0.875rem;
        }
        
        .text-success {
          color: #00a91c;
        }
        
        .text-error {
          color: #d54309;
        }
        
        .usa-table-container--scrollable {
          overflow-x: auto;
        }
      `}</style>
    </Layout>
  );
};

export default Trends;