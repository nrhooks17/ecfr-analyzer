import { useEffect, useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { api } from './api';
import LoadingScreen from './components/LoadingScreen';
import Dashboard from './components/Dashboard';
import Trends from './components/Trends';
import AgencyDetail from './components/AgencyDetail';
import Checksums from './components/Checksums';
import './App.css';

function App() {
  return (
    <Router>
      <div className="App">
        <Routes>
          <Route 
            path="/" 
            element={<Navigate to="/dashboard" replace />} 
          />
          <Route 
            path="/loading" 
            element={<LoadingScreen />} 
          />
          <Route 
            path="/dashboard" 
            element={<Dashboard />} 
          />
          <Route 
            path="/trends" 
            element={<Trends />} 
          />
          <Route 
            path="/agency/:slug" 
            element={<AgencyDetail />} 
          />
          <Route 
            path="/checksums" 
            element={<Checksums />} 
          />
          {/* Catch all route */}
          <Route 
            path="*" 
            element={<Navigate to="/dashboard" replace />} 
          />
        </Routes>
      </div>
    </Router>
  );
}

export default App;