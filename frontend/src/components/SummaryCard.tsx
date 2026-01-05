import React from 'react';

interface SummaryCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon?: string;
}

const SummaryCard: React.FC<SummaryCardProps> = ({ title, value, subtitle, icon }) => {
  const formatValue = (val: string | number): string => {
    if (typeof val === 'number') {
      if (val >= 1000000000) {
        return (val / 1000000000).toFixed(1) + 'B';
      } else if (val >= 1000000) {
        return (val / 1000000).toFixed(1) + 'M';
      } else if (val >= 1000) {
        return (val / 1000).toFixed(1) + 'K';
      }
      return val.toLocaleString();
    }
    return val;
  };

  return (
    <div className="usa-card">
      <div className="usa-card__container">
        <div className="usa-card__body">
          <div className="summary-card-content">
            {icon && (
              <div className="summary-card-icon">
                <span className={`fa fa-${icon}`} aria-hidden="true"></span>
              </div>
            )}
            <div className="summary-card-text">
              <div className="summary-card-inline">
                <h3 className="usa-card__heading font-heading-md margin-bottom-0">
                  {title}:
                </h3>
                <p className="summary-card-value font-heading-xl text-primary margin-bottom-0">
                  {formatValue(value)}
                </p>
              </div>
              {subtitle && (
                <p className="summary-card-subtitle font-body-sm text-base-dark margin-bottom-0">
                  {subtitle}
                </p>
              )}
            </div>
          </div>
        </div>
      </div>

      <style>{`
        .usa-card {
          border: 1px solid #c9c9c9;
          border-radius: 0.5rem;
          box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
          height: 100%;
        }
        
        .summary-card-content {
          display: flex;
          align-items: flex-start;
          gap: 1rem;
        }
        
        .summary-card-icon {
          flex-shrink: 0;
          width: 3rem;
          height: 3rem;
          display: flex;
          align-items: center;
          justify-content: center;
          background-color: #005ea2;
          color: white;
          border-radius: 50%;
          font-size: 1.25rem;
        }
        
        .summary-card-text {
          flex: 1;
          min-width: 0;
        }
        
        .summary-card-inline {
          display: flex;
          align-items: baseline;
          gap: 0.5rem;
        }
        
        .summary-card-value {
          color: #005ea2;
          font-weight: 700;
          line-height: 1.1;
        }
        
        .summary-card-subtitle {
          color: #71767a;
        }
        
        .usa-card__heading {
          color: #1b1b1b;
          margin-bottom: 0.5rem;
        }
      `}</style>
    </div>
  );
};

export default SummaryCard;