import { Navigate, Route, Routes } from 'react-router'
import { AdminLayout } from './components/AdminLayout'
import { DriverLayout } from './components/DriverLayout'
import { RequireRole } from './components/RequireRole'
import { DeliveriesPage } from './pages/admin/DeliveriesPage'
import { DeliveryPage } from './pages/admin/DeliveryPage'
import { DriversPage } from './pages/admin/DriversPage'
import { DriverDeliveriesPage } from './pages/driver/DriverDeliveriesPage'
import { DriverDeliveryPage } from './pages/driver/DriverDeliveryPage'
import { NewDeliveryPage } from './pages/admin/NewDeliveryPage'
import { LoginPage } from './pages/LoginPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { TrackingPage } from './pages/TrackingPage'
import { TrackingSearchPage } from './pages/TrackingSearchPage'

export function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/rastreio" replace />} />
      <Route path="/rastreio" element={<TrackingSearchPage />} />
      <Route path="/rastreio/:code" element={<TrackingPage />} />
      <Route path="/entrar" element={<LoginPage />} />
      <Route
        path="/admin"
        element={
          <RequireRole role="admin">
            <AdminLayout />
          </RequireRole>
        }
      >
        <Route index element={<Navigate to="entregas" replace />} />
        <Route path="entregas" element={<DeliveriesPage />} />
        <Route path="entregas/nova" element={<NewDeliveryPage />} />
        <Route path="entregas/:id" element={<DeliveryPage />} />
        <Route path="motoristas" element={<DriversPage />} />
      </Route>
      <Route
        path="/motorista"
        element={
          <RequireRole role="driver">
            <DriverLayout />
          </RequireRole>
        }
      >
        <Route index element={<DriverDeliveriesPage />} />
        <Route path="entregas/:id" element={<DriverDeliveryPage />} />
      </Route>
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
