import React, { useState, useMemo } from 'react';
import { Agency } from '../api';

interface AgencyTableProps {
  agencies: Agency[];
  onAgencyClick?: (agency: Agency) => void;
}

type SortField = 'name' | 'wordCount' | 'percentOfTotal' | 'titleCount';
type SortDirection = 'asc' | 'desc';

const AgencyTable: React.FC<AgencyTableProps> = ({ agencies, onAgencyClick }) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [sortField, setSortField] = useState<SortField>('wordCount');
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc');
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [currentPage, setCurrentPage] = useState(1);
  const itemsPerPage = 50;

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('desc');
    }
  };

  const toggleRowExpansion = (agencyId: string) => {
    const newExpanded = new Set(expandedRows);
    if (newExpanded.has(agencyId)) {
      newExpanded.delete(agencyId);
    } else {
      newExpanded.add(agencyId);
    }
    setExpandedRows(newExpanded);
  };

  const filteredAndSortedAgencies = useMemo(() => {
    let filtered = agencies.filter((agency: Agency) =>
      agency.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      agency.slug.toLowerCase().includes(searchTerm.toLowerCase())
    );

    filtered.sort((a: Agency, b: Agency) => {
      let aValue: string | number;
      let bValue: string | number;

      switch (sortField) {
        case 'name':
          aValue = a.name.toLowerCase();
          bValue = b.name.toLowerCase();
          break;
        case 'wordCount':
          aValue = a.wordCount;
          bValue = b.wordCount;
          break;
        case 'percentOfTotal':
          aValue = a.percentOfTotal;
          bValue = b.percentOfTotal;
          break;
        case 'titleCount':
          aValue = a.titleCount;
          bValue = b.titleCount;
          break;
        default:
          aValue = a.wordCount;
          bValue = b.wordCount;
      }

      if (typeof aValue === 'string' && typeof bValue === 'string') {
        return sortDirection === 'asc' 
          ? aValue.localeCompare(bValue)
          : bValue.localeCompare(aValue);
      } else {
        return sortDirection === 'asc' 
          ? (aValue as number) - (bValue as number)
          : (bValue as number) - (aValue as number);
      }
    });

    return filtered;
  }, [agencies, searchTerm, sortField, sortDirection]);

  const paginatedAgencies = useMemo(() => {
    const startIndex = (currentPage - 1) * itemsPerPage;
    const endIndex = startIndex + itemsPerPage;
    return filteredAndSortedAgencies.slice(startIndex, endIndex);
  }, [filteredAndSortedAgencies, currentPage, itemsPerPage]);

  const totalPages = Math.ceil(filteredAndSortedAgencies.length / itemsPerPage);

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    setExpandedRows(new Set()); // Close all expanded rows when changing pages
  };

  // Reset to page 1 when search changes
  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(e.target.value);
    setCurrentPage(1);
  };

  const getSortIcon = (field: SortField) => {
    if (sortField !== field) return '↕️';
    return sortDirection === 'asc' ? '↑' : '↓';
  };

  return (
    <div className="agency-table-container">
      {/* Search/Filter */}
      <div className="margin-bottom-3">
        <label htmlFor="agency-search" className="usa-label">
          Search agencies
        </label>
        <input
          id="agency-search"
          type="text"
          className="usa-input"
          placeholder="Search by agency name..."
          value={searchTerm}
          onChange={handleSearchChange}
        />
      </div>

      {/* Results count */}
      <p className="font-body-sm text-base-dark margin-bottom-2">
        Showing {(currentPage - 1) * itemsPerPage + 1} to {Math.min(currentPage * itemsPerPage, filteredAndSortedAgencies.length)} of {filteredAndSortedAgencies.length} agencies
        {filteredAndSortedAgencies.length !== agencies.length && (
          <span> (filtered from {agencies.length} total)</span>
        )}
      </p>

      {/* Table */}
      <div className="usa-table-container--scrollable">
        <table className="usa-table usa-table--striped">
          <thead>
            <tr>
              <th scope="col">Rank</th>
              <th 
                scope="col" 
                className="sortable-header"
                onClick={() => handleSort('name')}
              >
                Agency Name {getSortIcon('name')}
              </th>
              <th 
                scope="col" 
                className="sortable-header text-right"
                onClick={() => handleSort('wordCount')}
              >
                Word Count {getSortIcon('wordCount')}
              </th>
              <th 
                scope="col" 
                className="sortable-header text-right"
                onClick={() => handleSort('percentOfTotal')}
              >
                % of Total {getSortIcon('percentOfTotal')}
              </th>
              <th 
                scope="col" 
                className="sortable-header text-right"
                onClick={() => handleSort('titleCount')}
              >
                Titles {getSortIcon('titleCount')}
              </th>
              <th scope="col">Checksum</th>
              <th scope="col">Actions</th>
            </tr>
          </thead>
          <tbody>
            {paginatedAgencies.map((agency: Agency, index: number) => {
              const globalIndex = (currentPage - 1) * itemsPerPage + index;
              return (
              <React.Fragment key={agency.id}>
                <tr>
                  <th scope="row">#{globalIndex + 1}</th>
                  <td>
                    <strong>{agency.name}</strong>
                    <br />
                    <span className="font-body-sm text-base-dark">{agency.slug}</span>
                  </td>
                  <td className="text-right font-mono-sm">
                    {agency.wordCount.toLocaleString()}
                  </td>
                  <td className="text-right font-mono-sm">
                    {agency.percentOfTotal.toFixed(2)}%
                  </td>
                  <td className="text-right font-mono-sm">
                    {agency.titleCount}
                  </td>
                  <td className="checksum-cell">
                    {agency.checksum ? (
                      <span 
                        className="checksum-display font-mono-xs"
                        title={`Full checksum: ${agency.checksum}`}
                      >
                        {agency.checksum.substring(0, 8)}...
                      </span>
                    ) : (
                      <span className="text-base-light">No checksum</span>
                    )}
                  </td>
                  <td>
                    <button
                      type="button"
                      className="usa-button usa-button--outline usa-button--small"
                      onClick={() => toggleRowExpansion(agency.id)}
                    >
                      {expandedRows.has(agency.id) ? 'Hide Details' : 'Show Details'}
                    </button>
                    {onAgencyClick && (
                      <button
                        type="button"
                        className="usa-button usa-button--small margin-left-1"
                        onClick={() => onAgencyClick(agency)}
                      >
                        View Full Details
                      </button>
                    )}
                  </td>
                </tr>
                {expandedRows.has(agency.id) && (
                  <tr className="expanded-row">
                    <td colSpan={7}>
                      <div className="padding-2 bg-base-lightest">
                        <h4 className="margin-bottom-1">Quick Summary</h4>
                        <div className="grid-row grid-gap">
                          <div className="grid-col-6">
                            <p className="margin-bottom-05">
                              <strong>Total Word Count:</strong> {agency.wordCount.toLocaleString()}
                            </p>
                            <p className="margin-bottom-05">
                              <strong>Percentage of CFR:</strong> {agency.percentOfTotal.toFixed(2)}%
                            </p>
                          </div>
                          <div className="grid-col-6">
                            <p className="margin-bottom-05">
                              <strong>CFR Titles:</strong> {agency.titleCount}
                            </p>
                            <p className="margin-bottom-05">
                              <strong>Agency Slug:</strong> {agency.slug}
                            </p>
                          </div>
                        </div>
                      </div>
                    </td>
                  </tr>
                )}
              </React.Fragment>
            );
            })}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <nav aria-label="Pagination" className="usa-pagination margin-top-4">
          <ul className="usa-pagination__list">
            {/* Previous button */}
            <li className="usa-pagination__item usa-pagination__arrow">
              <button
                className={`usa-pagination__button usa-pagination__previous-page ${currentPage === 1 ? 'usa-pagination__button--disabled' : ''}`}
                aria-label="Previous page"
                disabled={currentPage === 1}
                onClick={() => handlePageChange(currentPage - 1)}
              >
                <span className="usa-pagination__link-text">Previous</span>
              </button>
            </li>

            {/* Page numbers */}
            {Array.from({ length: totalPages }, (_, i) => i + 1).map((page: number) => {
              // Show first page, last page, current page, and 2 pages around current
              const showPage = page === 1 || page === totalPages || 
                             (page >= currentPage - 2 && page <= currentPage + 2);
              
              if (!showPage) {
                // Show ellipsis for gaps
                if (page === currentPage - 3 || page === currentPage + 3) {
                  return (
                    <li key={`ellipsis-${page}`} className="usa-pagination__item usa-pagination__overflow">
                      <span>…</span>
                    </li>
                  );
                }
                return null;
              }

              return (
                <li key={page} className="usa-pagination__item usa-pagination__page-no">
                  <button
                    className={`usa-pagination__button ${page === currentPage ? 'usa-current' : ''}`}
                    aria-label={`Page ${page}`}
                    aria-current={page === currentPage ? 'page' : undefined}
                    onClick={() => handlePageChange(page)}
                  >
                    {page}
                  </button>
                </li>
              );
            })}

            {/* Next button */}
            <li className="usa-pagination__item usa-pagination__arrow">
              <button
                className={`usa-pagination__button usa-pagination__next-page ${currentPage === totalPages ? 'usa-pagination__button--disabled' : ''}`}
                aria-label="Next page"
                disabled={currentPage === totalPages}
                onClick={() => handlePageChange(currentPage + 1)}
              >
                <span className="usa-pagination__link-text">Next</span>
              </button>
            </li>
          </ul>
        </nav>
      )}

      <style>{`
        .sortable-header {
          cursor: pointer;
          user-select: none;
          transition: background-color 0.2s;
        }
        
        .sortable-header:hover {
          background-color: #f0f0f0;
        }
        
        .expanded-row {
          background-color: #f8f9fa;
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
        
        .usa-table-container--scrollable {
          overflow-x: auto;
        }
      `}</style>
    </div>
  );
};

export default AgencyTable;