import { Navigate, Outlet, Route, Routes } from 'react-router'
import { AuthProvider } from './components/AuthProvider'
import { CarrierLayout } from './components/CarrierLayout'
import { DriverLayout } from './components/DriverLayout'
import { RequireRole } from './components/RequireRole'
import { DeliveriesPage } from './pages/carrier/DeliveriesPage'
import { DeliveryPage } from './pages/carrier/DeliveryPage'
import { DriversPage } from './pages/carrier/DriversPage'
import { LabelPage } from './pages/carrier/LabelPage'
import { NewDeliveryPage } from './pages/carrier/NewDeliveryPage'
import { OverviewPage } from './pages/carrier/OverviewPage'
import { DriverDeliveriesPage } from './pages/driver/DriverDeliveriesPage'
import { DriverDeliveryPage } from './pages/driver/DriverDeliveryPage'
import { RoutePage } from './pages/driver/RoutePage'
import { LandingPage } from './pages/LandingPage'
import { LoginPage } from './pages/LoginPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { SignUpPage } from './pages/SignUpPage'
import { TrackingPage } from './pages/TrackingPage'
import { TrackingSearchPage } from './pages/TrackingSearchPage'

export function App() {
  return (
    <Routes>
      {/* Quem abre o rastreio é quem recebe a entrega, sem conta: estas telas
          não perguntam à API por uma sessão. */}
      <Route path="/rastreio" element={<TrackingSearchPage />} />
      <Route path="/rastreio/:code" element={<TrackingPage />} />
      <Route element={<WithSession />}>
        <Route path="/" element={<LandingPage />} />
        {/* Endereço antigo do login: a página inicial leva a cada área. */}
        <Route path="/entrar" element={<Navigate to="/" replace />} />
        <Route path="/transportadora/entrar" element={<LoginPage audience="carrier" />} />
        <Route path="/transportadora/cadastro" element={<SignUpPage />} />
        <Route path="/motorista/entrar" element={<LoginPage audience="driver" />} />
        <Route
          path="/transportadora"
          element={
            <RequireRole role="carrier">
              <CarrierLayout />
            </RequireRole>
          }
        >
          <Route index element={<OverviewPage />} />
          <Route path="entregas" element={<DeliveriesPage />} />
          <Route path="entregas/nova" element={<NewDeliveryPage />} />
          <Route path="entregas/:id" element={<DeliveryPage />} />
          <Route path="motoristas" element={<DriversPage />} />
        </Route>
        {/* Fora da moldura do painel: a página é só a etiqueta, pronta para imprimir. */}
        <Route
          path="/transportadora/entregas/:id/etiqueta"
          element={
            <RequireRole role="carrier">
              <LabelPage />
            </RequireRole>
          }
        />
        <Route
          path="/motorista"
          element={
            <RequireRole role="driver">
              <DriverLayout />
            </RequireRole>
          }
        >
          <Route index element={<DriverDeliveriesPage />} />
          <Route path="rota" element={<RoutePage />} />
          <Route path="entregas/:id" element={<DriverDeliveryPage />} />
        </Route>
      </Route>
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}

/**
 * As telas que dependem de quem está logado. Ao abrir uma delas, o cookie diz
 * à API se ainda existe uma sessão; a página inicial também, para levar quem
 * já entrou direto à sua área.
 */
function WithSession() {
  return (
    <AuthProvider>
      <Outlet />
    </AuthProvider>
  )
}
