import { Navigate, Route, Routes } from 'react-router'
import { NotFoundPage } from './pages/NotFoundPage'
import { TrackingPage } from './pages/TrackingPage'
import { TrackingSearchPage } from './pages/TrackingSearchPage'

export function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/rastreio" replace />} />
      <Route path="/rastreio" element={<TrackingSearchPage />} />
      <Route path="/rastreio/:code" element={<TrackingPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
