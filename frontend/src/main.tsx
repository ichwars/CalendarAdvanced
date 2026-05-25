import React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from './app/App';
import './app/styles.css';
import { applyInitialTheme } from './shared/theme';
import { applyGeneralPreferences } from './shared/preferences';

applyInitialTheme();
applyGeneralPreferences();

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
