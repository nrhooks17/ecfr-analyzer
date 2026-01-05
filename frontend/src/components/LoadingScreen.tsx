import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, ImportStatus } from '../api';
import Layout from './Layout';

const LoadingScreen: React.FC = () => {
  const [status, setStatus] = useState<ImportStatus | null>(null);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    const pollStatus = async () => {
      try {
        const currentStatus = await api.getStatus();
        setStatus(currentStatus);
        
        // If loading is complete, redirect to dashboard
        if (!currentStatus.isLoading && currentStatus.progress >= 100) {
          setTimeout(() => navigate('/dashboard'), 2000);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to get status');
      }
    };

    // Poll status every 2 seconds
    pollStatus();
    const interval = setInterval(pollStatus, 2000);

    return () => clearInterval(interval);
  }, [navigate]);

  if (error) {
    return (
      <Layout showNavigation={false}>
        <div className="usa-alert usa-alert--error">
          <div className="usa-alert__body">
            <h3 className="usa-alert__heading">Error Loading Data</h3>
            <p className="usa-alert__text">{error}</p>
          </div>
        </div>
      </Layout>
    );
  }

  if (!status) {
    return (
      <Layout showNavigation={false}>
        <div className="loading-screen">
          <div className="loading-content">
            <h1>eCFR Analyzer</h1>
            <p>Connecting to server...</p>
          </div>
        </div>
      </Layout>
    );
  }

  return (
    <Layout showNavigation={false}>
      <div className="loading-screen">
        <div className="loading-content">
          <h1 className="font-heading-2xl margin-bottom-3">eCFR Analyzer</h1>
          <h2 className="font-heading-lg margin-bottom-4">Loading Regulatory Data</h2>
          
          {/* Progress Bar */}
          <div className="usa-progress margin-bottom-4">
            <div className="usa-progress__bar" style={{ width: `${status.progress}%` }}></div>
          </div>
          
          {/* Status Information */}
          <div className="loading-status">
            <p className="font-body-lg margin-bottom-1">
              <strong>{status.currentStep}</strong>
            </p>
            <p className="font-body-md text-base-dark margin-bottom-2">
              {status.progress}% Complete
            </p>
            
            {status.totalTitles > 0 && (
              <p className="font-body-sm text-base">
                Processing {status.currentTitle} of {status.totalTitles} CFR titles
              </p>
            )}
            
            {status.error && (
              <div className="usa-alert usa-alert--warning margin-top-3">
                <div className="usa-alert__body">
                  <p className="usa-alert__text">{status.error}</p>
                </div>
              </div>
            )}
            
            <p className="font-body-xs text-base-light margin-top-4">
              Last updated: {new Date(status.lastUpdated).toLocaleTimeString()}
            </p>
          </div>
          
          {status.progress >= 100 && !status.isLoading && (
            <div className="usa-alert usa-alert--success margin-top-4">
              <div className="usa-alert__body">
                <p className="usa-alert__text">
                  Data loading complete! Redirecting to dashboard...
                </p>
              </div>
            </div>
          )}
        </div>
      </div>

      <style>{`
        .loading-screen {
          min-height: 60vh;
          display: flex;
          align-items: center;
          justify-content: center;
        }
        
        .loading-content {
          text-align: center;
          max-width: 600px;
          width: 100%;
        }
        
        .usa-progress {
          height: 1rem;
          background-color: #f0f0f0;
          border-radius: 0.25rem;
          overflow: hidden;
        }
        
        .usa-progress__bar {
          height: 100%;
          background-color: #005ea2;
          transition: width 0.5s ease;
        }
        
        .loading-status {
          background-color: #f8f9fa;
          padding: 2rem;
          border-radius: 0.5rem;
          border: 1px solid #dee2e6;
        }
      `}</style>
    </Layout>
  );
};

export default LoadingScreen;