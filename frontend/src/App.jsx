import React from 'react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import HostList from './pages/HostList';
// import HostDetail from './pages/HostDetail';

export default function App() {
    return (
        <BrowserRouter>
            <Routes>
                <Route path="/" element={<HostList />} />
                {/* <Route path="/host/:uuid" element={<HostDetail />} /> */}
            </Routes>
        </BrowserRouter>
    );
}