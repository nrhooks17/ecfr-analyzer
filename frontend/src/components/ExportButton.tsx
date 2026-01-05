import React, { useState } from 'react';
import { api } from '../api';

interface ExportButtonProps {
  type: 'agencies' | 'titles' | 'metrics';
  label?: string;
  className?: string;
}

const ExportButton: React.FC<ExportButtonProps> = ({ 
  type, 
  label,
  className = "usa-button usa-button--outline"
}) => {
  const [isExporting, setIsExporting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleExport = async () => {
    setIsExporting(true);
    setError(null);

    try {
      const blob = await api.exportData(type);
      
      // Create download link
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `ecfr-${type}-${new Date().toISOString().split('T')[0]}.json`;
      
      // Trigger download
      document.body.appendChild(link);
      link.click();
      
      // Cleanup
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (err) {
      console.error('Export failed:', err);
      setError(err instanceof Error ? err.message : 'Export failed');
    } finally {
      setIsExporting(false);
    }
  };

  const getDefaultLabel = () => {
    switch (type) {
      case 'agencies':
        return 'Export Agencies';
      case 'titles':
        return 'Export Titles';
      case 'metrics':
        return 'Export Metrics';
      default:
        return 'Export Data';
    }
  };

  return (
    <div className="export-button-container">
      <button
        type="button"
        className={className}
        onClick={handleExport}
        disabled={isExporting}
      >
        {isExporting ? (
          <>
            <span className="margin-right-05">⏳</span>
            Exporting...
          </>
        ) : (
          <>
            <span className="margin-right-05">📥</span>
            {label || getDefaultLabel()}
          </>
        )}
      </button>
      
      {error && (
        <div className="usa-alert usa-alert--error usa-alert--slim margin-top-1">
          <div className="usa-alert__body">
            <p className="usa-alert__text">{error}</p>
          </div>
        </div>
      )}
    </div>
  );
};

export default ExportButton;