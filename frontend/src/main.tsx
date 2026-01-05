import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import '@uswds/uswds/css/uswds.min.css'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)